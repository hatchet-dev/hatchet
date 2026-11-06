import {
  CONTROL_PLANE_TENANT_STORAGE_KEY,
  exchangeTokenInterceptor,
  fetchControlPlaneStatus,
  getApiErrorStatus,
  readTenantIdFromLocation,
  resolveExchangeTokenTenantId,
} from './api';
import { SupportGrantRole } from './generated/control-plane/data-contracts';
import { SUPPORT_SESSION_STORAGE_KEY } from '@/lib/support-session';
import { AxiosHeaders, InternalAxiosRequestConfig } from 'axios';
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

const supportSession = {
  token: 'support-token',
  apiUrl: 'https://shard.example.com',
  expiresAt: '2099-01-01T00:00:00Z',
  grant: {
    id: '44444444-4444-4444-8444-444444444444',
    tenantId: tenantA,
    tenantName: 'Tenant A',
    tenantSlug: 'tenant-a',
    role: SupportGrantRole.SUPPORT_VIEWER,
    reason: 'incident',
    staffEmail: 'staff@example.com',
    createdAt: '2099-01-01T00:00:00Z',
    expiresAt: '2099-01-02T00:00:00Z',
  },
};

async function withSupportSession(fn: () => Promise<void>) {
  const previousSessionStorage = globalThis.sessionStorage;

  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    value: {
      getItem: (key: string) =>
        key === SUPPORT_SESSION_STORAGE_KEY
          ? JSON.stringify({ session: supportSession })
          : null,
      setItem: () => {},
      removeItem: () => {},
    },
  });

  try {
    await fn();
  } finally {
    if (previousSessionStorage) {
      Object.defineProperty(globalThis, 'sessionStorage', {
        configurable: true,
        value: previousSessionStorage,
      });
    } else {
      delete (globalThis as Partial<typeof globalThis>).sessionStorage;
    }
  }
}

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

test('exchangeTokenInterceptor routes tenant requests to the support shard with the support token', async () => {
  await withSupportSession(async () => {
    const config = await exchangeTokenInterceptor(
      requestConfig({
        url: `/api/v1/tenants/${tenantA}`,
        xResources: ['tenant'],
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
  });
});

test('exchangeTokenInterceptor routes non-tenant requests to the support shard too', async () => {
  await withSupportSession(async () => {
    const config = await exchangeTokenInterceptor(
      requestConfig({ xResources: [] }),
    );

    assert.equal(config.baseURL, supportSession.apiUrl);
    assert.equal(config.withCredentials, false);
    assert.equal(
      config.headers.get('Authorization'),
      `Bearer ${supportSession.token}`,
    );
    assert.equal(config.headers.get('X-Support-Token'), 'true');
  });
});

test('exchangeTokenInterceptor ignores the page tenant in support mode', async () => {
  await withSupportSession(async () => {
    await new Promise<void>((resolve, reject) => {
      withLocation(`/tenants/${tenantB}/runs`, () => {
        exchangeTokenInterceptor(
          requestConfig({
            url: `/api/v1/tenants/${tenantB}`,
            xResources: ['tenant'],
          }),
        )
          .then((config) => {
            assert.equal(config.baseURL, supportSession.apiUrl);
            assert.equal(
              config.headers.get('Authorization'),
              `Bearer ${supportSession.token}`,
            );
            resolve();
          })
          .catch(reject);
      });
    });
  });
});

test('exchangeTokenInterceptor leaves non-tenant requests alone without a support session', async () => {
  const config = await exchangeTokenInterceptor(
    requestConfig({ xResources: [] }),
  );

  assert.equal(config.baseURL, '');
  assert.equal(config.withCredentials, true);
  assert.equal(config.headers.get('Authorization'), undefined);
  assert.equal(config.headers.get('X-Support-Token'), undefined);
});

test('fetchControlPlaneStatus reports the control plane disabled in support mode', async () => {
  await withSupportSession(async () => {
    const status = await fetchControlPlaneStatus();

    assert.equal(status.isControlPlaneEnabled, false);
    assert.equal(status.meta, null);
  });
});
