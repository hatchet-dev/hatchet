import {
  SupportGrant,
  SupportGrantRole,
} from '@/lib/api/generated/control-plane/data-contracts';
import {
  clearEndedSupportGrants,
  findSupportGrant,
  getEndedSupportGrant,
  markSupportSessionEnded,
  resolveSupportSession,
  supportRoleLabel,
  supportTokenQueryOptions,
} from '@/lib/support-session';
import { AxiosError, AxiosHeaders, InternalAxiosRequestConfig } from 'axios';
import assert from 'node:assert/strict';
import { afterEach, test } from 'node:test';

const tenantA = '11111111-1111-4111-8111-111111111111';
const tenantB = '22222222-2222-4222-8222-222222222222';

const grant: SupportGrant = {
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

const session = {
  token: 'support-token',
  apiUrl: 'https://shard.example.com',
  expiresAt: '2099-01-01T00:00:00Z',
  grant,
};

function httpError(status: number) {
  return new AxiosError(
    'rejected',
    'ERR_BAD_REQUEST',
    { headers: new AxiosHeaders() } as InternalAxiosRequestConfig,
    null,
    {
      status,
      statusText: '',
      headers: {},
      config: { headers: new AxiosHeaders() } as InternalAxiosRequestConfig,
      data: {},
    },
  );
}

afterEach(() => {
  clearEndedSupportGrants();
});

test('findSupportGrant matches on the tenant', () => {
  assert.equal(findSupportGrant([grant], tenantA), grant);
  assert.equal(findSupportGrant([grant], tenantB), null);
  assert.equal(findSupportGrant(undefined, tenantA), null);
  assert.equal(findSupportGrant([grant], undefined), null);
});

test('resolveSupportSession reports a live grant', () => {
  assert.deepEqual(resolveSupportSession(tenantA, [grant]), {
    grant,
    isSupport: true,
    isExpired: false,
  });
  assert.deepEqual(resolveSupportSession(tenantB, [grant]), {
    grant: null,
    isSupport: false,
    isExpired: false,
  });
});

test('an ended grant stays visible as expired until a newer grant replaces it', () => {
  markSupportSessionEnded(grant);

  assert.deepEqual(resolveSupportSession(tenantA, []), {
    grant,
    isSupport: true,
    isExpired: true,
  });
  assert.deepEqual(resolveSupportSession(tenantA, [grant]), {
    grant,
    isSupport: true,
    isExpired: true,
  });

  const renewed = { ...grant, id: '55555555-5555-4555-8555-555555555555' };
  assert.deepEqual(resolveSupportSession(tenantA, [renewed]), {
    grant: renewed,
    isSupport: true,
    isExpired: false,
  });
});

test('supportTokenQueryOptions marks the grant ended when the control plane refuses a token', async () => {
  const refused = supportTokenQueryOptions(grant, () =>
    Promise.reject(httpError(403)),
  );
  await assert.rejects(refused.queryFn());
  assert.equal(getEndedSupportGrant(tenantA), grant);

  clearEndedSupportGrants();

  const flaky = supportTokenQueryOptions(grant, () =>
    Promise.reject(httpError(503)),
  );
  await assert.rejects(flaky.queryFn());
  assert.equal(getEndedSupportGrant(tenantA), null);
});

test('supportTokenQueryOptions goes stale a minute before the token expires', async () => {
  const options = supportTokenQueryOptions(grant, () =>
    Promise.resolve(session),
  );
  assert.equal(await options.queryFn(), session);

  const dataUpdatedAt = new Date('2098-12-31T23:50:00Z').getTime();
  assert.equal(
    options.staleTime({ state: { data: session, dataUpdatedAt } } as never),
    9 * 60_000,
  );
  assert.equal(
    options.staleTime({ state: { data: undefined, dataUpdatedAt } } as never),
    0,
  );
});

test('supportRoleLabel maps grant roles to banner labels', () => {
  assert.equal(supportRoleLabel(SupportGrantRole.SUPPORT_VIEWER), 'viewer');
  assert.equal(supportRoleLabel(SupportGrantRole.SUPPORT_OPERATOR), 'operator');
});
