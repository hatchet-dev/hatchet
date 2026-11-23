import { controlPlaneApi, exchangeTokenInterceptor } from './api';
import { exchangeTokenIsUsable } from './exchange-token';
import { TenantExchangeToken } from './generated/control-plane/data-contracts';
import queryClient from '@/query-client';
import { AxiosHeaders, InternalAxiosRequestConfig } from 'axios';
import assert from 'node:assert/strict';
import { after, test } from 'node:test';

const MINUTE_MS = 60_000;

// Cached tokens hold an hour-long garbage-collection timer that would keep
// the test process alive.
after(() => queryClient.clear());

function tokenExpiringIn(name: string, ms: number): TenantExchangeToken {
  return {
    token: name,
    apiUrl: 'https://shard.example',
    expiresAt: new Date(Date.now() + ms).toISOString(),
  };
}

function tenantRequest(tenantId: string): InternalAxiosRequestConfig {
  return {
    url: `/api/v1/tenants/${tenantId}/workflows`,
    xResources: ['tenant'],
    xTenantId: tenantId,
    headers: new AxiosHeaders(),
  };
}

// Serves the given tokens in order, repeating the last one, and counts calls.
function stubControlPlane(tokens: TenantExchangeToken[]) {
  const calls = { count: 0 };

  controlPlaneApi.metadataGet = (async () => ({
    data: {},
  })) as unknown as typeof controlPlaneApi.metadataGet;

  controlPlaneApi.exchangeTokenCreate = (async () => {
    const data = tokens[Math.min(calls.count, tokens.length - 1)];
    calls.count += 1;
    return { data };
  }) as unknown as typeof controlPlaneApi.exchangeTokenCreate;

  return calls;
}

function bearer(config: InternalAxiosRequestConfig) {
  return config.headers.get('Authorization');
}

test('exchange token is usable only outside the expiry buffer', () => {
  const now = Date.parse('2026-01-01T00:00:00Z');

  assert.equal(
    exchangeTokenIsUsable(new Date(now + MINUTE_MS + 1_000).toISOString(), now),
    true,
  );
  assert.equal(
    exchangeTokenIsUsable(new Date(now + MINUTE_MS).toISOString(), now),
    false,
  );
  assert.equal(exchangeTokenIsUsable('not-a-date', now), false);
});

test('interceptor reuses a cached token that is still usable', async () => {
  const tenantId = '11111111-1111-4111-8111-111111111111';
  const calls = stubControlPlane([tokenExpiringIn('fresh', 5 * MINUTE_MS)]);

  const first = await exchangeTokenInterceptor(tenantRequest(tenantId));
  const second = await exchangeTokenInterceptor(tenantRequest(tenantId));

  assert.equal(bearer(first), 'Bearer fresh');
  assert.equal(bearer(second), 'Bearer fresh');
  assert.equal(calls.count, 1);
});

test('interceptor replaces a token that arrives already expired', async () => {
  const tenantId = '22222222-2222-4222-8222-222222222222';
  const calls = stubControlPlane([
    tokenExpiringIn('late', -10 * MINUTE_MS),
    tokenExpiringIn('fresh', 5 * MINUTE_MS),
  ]);

  const config = await exchangeTokenInterceptor(tenantRequest(tenantId));

  assert.equal(bearer(config), 'Bearer fresh');
  assert.equal(calls.count, 2);
});

test('requests waiting on one late token share a single replacement', async () => {
  const tenantId = '33333333-3333-4333-8333-333333333333';
  const calls = stubControlPlane([
    tokenExpiringIn('late', -10 * MINUTE_MS),
    tokenExpiringIn('fresh', 5 * MINUTE_MS),
  ]);

  const configs = await Promise.all(
    [1, 2, 3, 4].map(() => exchangeTokenInterceptor(tenantRequest(tenantId))),
  );

  assert.deepEqual(configs.map(bearer), [
    'Bearer fresh',
    'Bearer fresh',
    'Bearer fresh',
    'Bearer fresh',
  ]);
  assert.equal(calls.count, 2);
});

test('interceptor still sends when the replacement also looks expired', async () => {
  // A client clock running fast makes every token look expired. The API
  // decides, so the request must go out instead of failing locally.
  const tenantId = '44444444-4444-4444-8444-444444444444';
  const calls = stubControlPlane([tokenExpiringIn('skewed', -MINUTE_MS)]);

  const config = await exchangeTokenInterceptor(tenantRequest(tenantId));

  assert.equal(bearer(config), 'Bearer skewed');
  assert.equal(calls.count, 2);
});
