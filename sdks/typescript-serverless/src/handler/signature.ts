/**
 * Request signatures, WebCrypto only. The operator signs every POST body with
 * hex(hmac_sha256(secret, body)) (internal/signature/sign.go) and the bodyless websocket
 * upgrade with the same digest over `upgradeSigningPayload`.
 */
import {
  ENDPOINT_ID_HEADER,
  INVOCATION_HEADER,
  NONCE_HEADER,
  SIGNATURE_HEADER,
  TASK_ID_HEADER,
  TIMESTAMP_HEADER,
  UPGRADE_MAX_AGE_SECONDS,
  upgradeSigningPayload,
} from './contract';

const encoder = new TextEncoder();

/** hex(hmac_sha256(secret, data)), the digest internal/signature.Sign produces. */
export async function signHex(secret: string, data: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    'raw',
    encoder.encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  );
  const mac = await crypto.subtle.sign('HMAC', key, encoder.encode(data));

  return toHex(new Uint8Array(mac));
}

function toHex(bytes: Uint8Array): string {
  let out = '';

  for (const b of bytes) {
    out += b.toString(16).padStart(2, '0');
  }

  return out;
}

/**
 * Constant-time string comparison. Workers expose the non-standard
 * crypto.subtle.timingSafeEqual; the fallback never short-circuits on the first mismatch.
 */
export function constantTimeEqual(a: string, b: string): boolean {
  const ab = encoder.encode(a);
  const bb = encoder.encode(b);

  if (ab.byteLength !== bb.byteLength) {
    return false;
  }

  const subtle = crypto.subtle as SubtleCrypto & {
    timingSafeEqual?: (x: ArrayBufferView, y: ArrayBufferView) => boolean;
  };

  if (typeof subtle.timingSafeEqual === 'function') {
    return subtle.timingSafeEqual(ab, bb);
  }

  let diff = 0;

  for (let i = 0; i < ab.length; i++) {
    diff |= ab[i] ^ bb[i];
  }

  return diff === 0;
}

/**
 * Verifies X-Hatchet-Signature on a POST (healthcheck or non-durable trigger). The signature
 * covers the raw body bytes exactly as sent, so read the body as text before parsing it.
 */
export async function verifyBodySignature(
  body: string,
  header: string | null | undefined,
  secret: string
): Promise<boolean> {
  if (!header || !secret) {
    return false;
  }

  const expected = await signHex(secret, body);

  return constantTimeEqual(expected, header);
}

export type UpgradeVerification =
  | { ok: true; endpointId: string; taskId: string; invocation: number; nonce: string }
  | { ok: false; status: 401 | 403; reason: string };

export interface VerifyUpgradeOptions {
  /** When set, upgrades carrying another endpoint id are refused with 403. */
  endpointId?: string;
  /** The current time in unix seconds; defaults to the wall clock. */
  nowSeconds?: number;
  /** A nonce set for replay protection; returns true when the nonce was seen before. */
  seenNonce?: (nonce: string) => boolean;
}

/**
 * Verifies the bodyless websocket upgrade (pkg/serverlessoperator/durable/dial.go). The
 * signature covers `upgradeSigningPayload`; timestamps older than UpgradeMaxAge are
 * rejected. Used by the durable relay; exported so adapters can verify before accepting.
 */
export async function verifyUpgradeSignature(
  headers: Headers,
  secret: string,
  opts: VerifyUpgradeOptions = {}
): Promise<UpgradeVerification> {
  const endpointId = headers.get(ENDPOINT_ID_HEADER) ?? '';

  if (opts.endpointId && endpointId !== opts.endpointId) {
    return { ok: false, status: 403, reason: 'unknown endpoint id' };
  }

  const timestamp = headers.get(TIMESTAMP_HEADER) ?? '';
  const ts = Number.parseInt(timestamp, 10);
  const now = opts.nowSeconds ?? Math.floor(Date.now() / 1000);

  if (!Number.isFinite(ts) || now - ts > UPGRADE_MAX_AGE_SECONDS) {
    return { ok: false, status: 401, reason: 'stale or missing timestamp' };
  }

  const nonce = headers.get(NONCE_HEADER) ?? '';

  if (nonce === '' || (opts.seenNonce && opts.seenNonce(nonce))) {
    return { ok: false, status: 401, reason: 'missing or replayed nonce' };
  }

  const taskId = headers.get(TASK_ID_HEADER) ?? '';
  const invocation = headers.get(INVOCATION_HEADER) ?? '';
  const expected = await signHex(
    secret,
    upgradeSigningPayload(timestamp, nonce, taskId, invocation)
  );

  if (!constantTimeEqual(expected, headers.get(SIGNATURE_HEADER) ?? '')) {
    return { ok: false, status: 401, reason: 'bad signature' };
  }

  return { ok: true, endpointId, taskId, invocation: Number.parseInt(invocation, 10), nonce };
}
