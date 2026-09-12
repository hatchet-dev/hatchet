import { describe, expect, it } from 'vitest';
import {
  ENDPOINT_ID_HEADER,
  INVOCATION_HEADER,
  NONCE_HEADER,
  SIGNATURE_HEADER,
  TASK_ID_HEADER,
  TIMESTAMP_HEADER,
  upgradeSigningPayload,
} from './contract';
import {
  constantTimeEqual,
  signHex,
  verifyBodySignature,
  verifyUpgradeSignature,
} from './signature';

const secret = 'test-secret-at-least-32-characters-long';

describe('body signatures', () => {
  it('produces the hex HMAC-SHA256 digest internal/signature.Sign produces', async () => {
    // echo -n 'hello' | openssl dgst -sha256 -hmac 'key'
    expect(await signHex('key', 'hello')).toBe(
      '9307b3b915efb5171ff14d8cb55fbcc798c6c0ef1456d66ded1a6aa723a58b7b'
    );
  });

  it('round-trips a signed body', async () => {
    const body = '{"endpointId":"e","namespace":"n","timestamp":"1"}';
    const signature = await signHex(secret, body);

    expect(await verifyBodySignature(body, signature, secret)).toBe(true);
  });

  it('rejects a wrong secret, a tampered body and a missing header', async () => {
    const body = '{"a":1}';
    const signature = await signHex(secret, body);

    expect(await verifyBodySignature(body, signature, 'another-secret')).toBe(false);
    expect(await verifyBodySignature('{"a":2}', signature, secret)).toBe(false);
    expect(await verifyBodySignature(body, null, secret)).toBe(false);
    expect(await verifyBodySignature(body, '', secret)).toBe(false);
    expect(await verifyBodySignature(body, signature, '')).toBe(false);
  });

  it('compares in constant time without leaking on length', () => {
    expect(constantTimeEqual('abc', 'abc')).toBe(true);
    expect(constantTimeEqual('abc', 'abd')).toBe(false);
    expect(constantTimeEqual('abc', 'abcd')).toBe(false);
  });
});

describe('upgrade signatures', () => {
  const now = 1_700_000_000;

  it('signs the same bytes as the operator (pkg/serverlessoperator/durable/dial.go)', async () => {
    // Reference values from contract.UpgradeSigningPayload and internal/signature.Sign on
    // this tree, for endpoint "endpoint-1", timestamp 1700000000, nonce "nonce-1", task
    // "task-1", invocation 2 and the secret below.
    const payload = upgradeSigningPayload('endpoint-1', '1700000000', 'nonce-1', 'task-1', '2');

    expect(payload).toBe('endpoint-1.1700000000.nonce-1.task-1.2');
    expect(await signHex(secret, payload)).toBe(
      '65ee3666e7f124ccb9fa5cba9ff10ba03d84c47d6179690617aef2c010ef496d'
    );
  });

  it('refuses an upgrade without an endpoint id header', async () => {
    const headers = await signedHeaders();
    headers.delete(ENDPOINT_ID_HEADER);

    expect(await verifyUpgradeSignature(headers, secret, { nowSeconds: now })).toMatchObject({
      ok: false,
      status: 401,
      reason: 'missing endpoint id',
    });
  });

  it('refuses a signature made for another endpoint id', async () => {
    const headers = await signedHeaders();
    headers.set(ENDPOINT_ID_HEADER, 'endpoint-2');

    expect(await verifyUpgradeSignature(headers, secret, { nowSeconds: now })).toMatchObject({
      ok: false,
      status: 401,
      reason: 'bad signature',
    });
  });

  it('refuses the upgrade with 503 when the nonce store is full', async () => {
    const result = await verifyUpgradeSignature(await signedHeaders(), secret, {
      nowSeconds: now,
      consumeNonce: () => 'full',
    });

    expect(result).toMatchObject({ ok: false, status: 503, reason: 'nonce store full' });
  });

  async function signedHeaders(overrides: Record<string, string> = {}, signWith = secret) {
    const values: Record<string, string> = {
      [TIMESTAMP_HEADER]: String(now - 10),
      [NONCE_HEADER]: 'nonce-1',
      [TASK_ID_HEADER]: 'task-1',
      [INVOCATION_HEADER]: '2',
      [ENDPOINT_ID_HEADER]: 'endpoint-1',
      ...overrides,
    };
    const payload = upgradeSigningPayload(
      values[ENDPOINT_ID_HEADER],
      values[TIMESTAMP_HEADER],
      values[NONCE_HEADER],
      values[TASK_ID_HEADER],
      values[INVOCATION_HEADER]
    );
    const headers = new Headers(values);
    headers.set(SIGNATURE_HEADER, await signHex(signWith, payload));

    return headers;
  }

  it('accepts a fresh, well-signed upgrade', async () => {
    const result = await verifyUpgradeSignature(await signedHeaders(), secret, { nowSeconds: now });

    expect(result).toEqual({
      ok: true,
      endpointId: 'endpoint-1',
      taskId: 'task-1',
      invocation: 2,
      nonce: 'nonce-1',
    });
  });

  it('rejects a timestamp older than five minutes', async () => {
    const headers = await signedHeaders({ [TIMESTAMP_HEADER]: String(now - 301) });
    const result = await verifyUpgradeSignature(headers, secret, { nowSeconds: now });

    expect(result).toMatchObject({ ok: false, status: 401, reason: 'stale or missing timestamp' });
  });

  it('rejects a bad signature, a replayed nonce and a foreign endpoint id', async () => {
    expect(
      await verifyUpgradeSignature(await signedHeaders({}, 'other'), secret, { nowSeconds: now })
    ).toMatchObject({ ok: false, status: 401, reason: 'bad signature' });

    expect(
      await verifyUpgradeSignature(await signedHeaders(), secret, {
        nowSeconds: now,
        consumeNonce: () => 'replayed',
      })
    ).toMatchObject({ ok: false, status: 401, reason: 'missing or replayed nonce' });

    expect(
      await verifyUpgradeSignature(await signedHeaders(), secret, {
        nowSeconds: now,
        endpointId: 'endpoint-2',
      })
    ).toMatchObject({ ok: false, status: 403, reason: 'unknown endpoint id' });
  });
});
