import {
  CONTROL_PLANE_TENANT_STORAGE_KEY,
  controlPlaneApi,
  exchangeTokenInterceptor,
  getApiErrorStatus,
  readTenantIdFromLocation,
  controlPlaneMetaQuery,
  resolveExchangeTokenTenantId,
  supportTokenRejected,
} from './api';
import {
  SupportGrant,
  SupportGrantRole,
} from './generated/control-plane/data-contracts';
import {
  clearEndedSupportGrants,
  SUPPORT_GRANTS_QUERY_KEY,
  supportTokenQueryKey,
} from '@/lib/support-session';
import queryClient from '@/query-client';
import { AxiosError, AxiosHeaders, InternalAxiosRequestConfig } from 'axios';
import assert from 'node:assert/strict';
import { test } from 'node:test';

const tenantA = '11111111-1111-4111-8111-111111111111';
const tenantB = '22222222-2222-4222-8222-222222222222';
const tenantC = '33333333-3333-4333-8333-333333333333';

function withStoredTenant(tenantId: string, fn: () => void) {
  const previousLocalStorage = globalThis.localStorage;

  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      getItem: (key: string) =>
        key === CONTROL_PLANE_TENANT_STORAGE_KEY
          ? JSON.stringify({ metadata: { id: tenantId } })
          : null,
    },
  });

  try {
    fn();
  } finally {
    if (previousLocalStorage) {
      Object.defineProperty(globalThis, 'localStorage', {
        configurable: true,
        value: previousLocalStorage,
      });
    } else {
      delete (globalThis as Partial<typeof globalThis>).localStorage;
    }
  }
}

function withLocation(pathname: string, fn: () => void) {
  const previousWindow = globalThis.window;

  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: {
      location: { pathname },
    },
  });

  try {
    fn();
  } finally {
    if (previousWindow) {
      Object.defineProperty(globalThis, 'window', {
        configurable: true,
        value: previousWindow,
      });
    } else {
      delete (globalThis as Partial<typeof globalThis>).window;
    }
  }
}

test('readTenantIdFromLocation parses tenant page paths', () => {
  withLocation(`/tenants/${tenantA}/runs`, () => {
    assert.equal(readTenantIdFromLocation(), tenantA);
  });
});

test('readTenantIdFromLocation parses tenant settings paths', () => {
  withLocation(`/tenants/${tenantA}/settings/members`, () => {
    assert.equal(readTenantIdFromLocation(), tenantA);
  });
});

test('readTenantIdFromLocation returns null on organization pages', () => {
  withLocation('/organizations/5b9f0665-ad27-4b5c-bf46-cfad1a280b66', () => {
    assert.equal(readTenantIdFromLocation(), null);
  });
});

test('readTenantIdFromLocation ignores non-UUID tenant segments', () => {
  withLocation('/tenants/not-a-uuid/runs', () => {
    assert.equal(readTenantIdFromLocation(), null);
  });
});

test('resolveExchangeTokenTenantId prefers explicit tenant over location and storage', () => {
  withLocation(`/tenants/${tenantB}/runs`, () => {
    withStoredTenant(tenantC, () => {
      assert.equal(
        resolveExchangeTokenTenantId({
          xTenantId: tenantA,
        }),
        tenantA,
      );
    });
  });
});

test('resolveExchangeTokenTenantId prefers location tenant over storage', () => {
  withLocation(`/tenants/${tenantA}/runs`, () => {
    withStoredTenant(tenantB, () => {
      assert.equal(resolveExchangeTokenTenantId({}), tenantA);
    });
  });
});

test('resolveExchangeTokenTenantId falls back to storage on organization pages', () => {
  withLocation('/organizations/5b9f0665-ad27-4b5c-bf46-cfad1a280b66', () => {
    withStoredTenant(tenantA, () => {
      assert.equal(resolveExchangeTokenTenantId({}), tenantA);
    });
  });
});

test('resolveExchangeTokenTenantId uses location tenant instead of stale storage', () => {
  withLocation(`/tenants/${tenantA}/runs`, () => {
    withStoredTenant(tenantC, () => {
      assert.equal(resolveExchangeTokenTenantId({}), tenantA);
    });
  });
});

test('getApiErrorStatus returns the HTTP response status', () => {
  assert.equal(getApiErrorStatus({ response: { status: 401 } }), 401);
  assert.equal(getApiErrorStatus({ status: 403 }), 403);
});

test('getApiErrorStatus ignores non-HTTP errors', () => {
  assert.equal(getApiErrorStatus(new Error('boom')), undefined);
  assert.equal(getApiErrorStatus({ response: { status: '401' } }), undefined);
});

const supportGrant: SupportGrant = {
  id: '44444444-4444-4444-8444-444444444444',
  tenantId: tenantA,
  tenantName: 'Tenant A',
  tenantSlug: 'tenant-a',
  role: SupportGrantRole.SUPPORT_VIEWER,
  reason: 'incident',
  staffEmail: 'staff@example.com',
  createdAt: '2099-01-01T00:00:00Z',
  expiresAt: '2099-01-02T00:00:00Z',
};

const supportSession = {
  token: 'support-token',
  apiUrl: 'https://shard.example.com',
  expiresAt: '2099-01-01T00:00:00Z',
  grant: supportGrant,
};

const exchangeToken = {
  token: 'exchange-token',
  apiUrl: 'https://east.example.com',
  expiresAt: '2099-01-01T00:00:00Z',
};

function requestConfig(
  overrides: Partial<InternalAxiosRequestConfig> = {},
): InternalAxiosRequestConfig {
  return {
    url: '/api/v1/users/current',
    baseURL: '',
    withCredentials: true,
    headers: new AxiosHeaders(),
    ...overrides,
  };
}

// A signed-in staff member under the control plane, with one support grant
async function withSupportGrant(fn: () => Promise<void>) {
  const originalToken = controlPlaneApi.supportSessionToken;
  const originalExchange = controlPlaneApi.exchangeTokenCreate;
  queryClient.clear();
  clearEndedSupportGrants();
  queryClient.setQueryData(controlPlaneMetaQuery.queryKey, {
    data: { canBill: false },
  } as never);
  queryClient.setQueryData(SUPPORT_GRANTS_QUERY_KEY, [supportGrant]);

  try {
    await fn();
  } finally {
    controlPlaneApi.supportSessionToken = originalToken;
    controlPlaneApi.exchangeTokenCreate = originalExchange;
    queryClient.clear();
    clearEndedSupportGrants();
  }
}

test('exchangeTokenInterceptor sends a support token to the shard of a tenant under a grant', async () => {
  await withSupportGrant(async () => {
    const seen: string[] = [];
    controlPlaneApi.supportSessionToken = (async (body: {
      tenantId: string;
    }) => {
      seen.push(body.tenantId);
      return { data: supportSession };
    }) as typeof controlPlaneApi.supportSessionToken;

    const config = await exchangeTokenInterceptor(
      requestConfig({
        url: `/api/v1/tenants/${tenantA}`,
        xResources: ['tenant'],
        xTenantId: tenantA,
      }),
    );

    assert.equal(config.baseURL, supportSession.apiUrl);
    assert.equal(config.withCredentials, false);
    assert.equal(
      config.headers.get('Authorization'),
      `Bearer ${supportSession.token}`,
    );
    assert.equal(config.headers.get('X-Support-Token'), 'true');
    assert.equal(config.headers.get('X-Exchange-Token'), undefined);

    // the token is cached until it nears expiry
    await exchangeTokenInterceptor(
      requestConfig({
        url: `/api/v1/tenants/${tenantA}/workflows`,
        xResources: ['tenant'],
        xTenantId: tenantA,
      }),
    );
    assert.deepEqual(seen, [tenantA]);
  });
});

test('exchangeTokenInterceptor keeps using exchange tokens for tenants without a grant', async () => {
  await withSupportGrant(async () => {
    controlPlaneApi.supportSessionToken = (async () => {
      throw new Error('not expected');
    }) as typeof controlPlaneApi.supportSessionToken;
    controlPlaneApi.exchangeTokenCreate = (async () => ({
      data: exchangeToken,
    })) as unknown as typeof controlPlaneApi.exchangeTokenCreate;

    const config = await exchangeTokenInterceptor(
      requestConfig({
        url: `/api/v1/tenants/${tenantB}`,
        xResources: ['tenant'],
        xTenantId: tenantB,
      }),
    );

    assert.equal(config.baseURL, exchangeToken.apiUrl);
    assert.equal(
      config.headers.get('Authorization'),
      `Bearer ${exchangeToken.token}`,
    );
    assert.equal(config.headers.get('X-Exchange-Token'), 'true');
    assert.equal(config.headers.get('X-Support-Token'), undefined);
  });
});

test('exchangeTokenInterceptor leaves non-tenant requests alone', async () => {
  await withSupportGrant(async () => {
    const config = await exchangeTokenInterceptor(
      requestConfig({ xResources: [] }),
    );

    assert.equal(config.baseURL, '');
    assert.equal(config.withCredentials, true);
    assert.equal(config.headers.get('Authorization'), undefined);
    assert.equal(config.headers.get('X-Support-Token'), undefined);
  });
});

function rejection(status: number, headers: Record<string, string>) {
  const config = {
    xTenantId: tenantA,
    headers: new AxiosHeaders(headers),
  } as InternalAxiosRequestConfig;
  return new AxiosError('rejected', 'ERR_BAD_REQUEST', config, null, {
    status,
    statusText: '',
    headers: {},
    config,
    data: {},
  });
}

test('supportTokenRejected drops a support token the shard refused', async () => {
  await withSupportGrant(async () => {
    queryClient.setQueryData(supportTokenQueryKey(tenantA), supportSession);

    await assert.rejects(
      supportTokenRejected(rejection(403, { 'X-Support-Token': 'true' })),
    );
    assert.equal(
      queryClient.getQueryData(supportTokenQueryKey(tenantA)),
      undefined,
    );
  });
});

test('supportTokenRejected ignores other failures', async () => {
  await withSupportGrant(async () => {
    queryClient.setQueryData(supportTokenQueryKey(tenantA), supportSession);

    await assert.rejects(
      supportTokenRejected(rejection(500, { 'X-Support-Token': 'true' })),
    );
    await assert.rejects(
      supportTokenRejected(rejection(403, { 'X-Exchange-Token': 'true' })),
    );
    await assert.rejects(supportTokenRejected(new Error('boom')));
    assert.equal(
      queryClient.getQueryData(supportTokenQueryKey(tenantA)),
      supportSession,
    );
  });
});
