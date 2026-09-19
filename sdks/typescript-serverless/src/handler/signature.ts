/**
 * Request verification, WebCrypto only. The operator signs every POST body with
 * hex(hmac_sha256(secret, body)) (internal/signature/sign.go) and the bodyless websocket
 * upgrade with the same digest over `upgradeSigningPayload`. Both carry a timestamp the
 * signature covers, which must lie within `REQUEST_MAX_AGE_SECONDS` of the endpoint's clock
 * in either direction, so a captured request cannot be replayed later.
 */
import {
  ENDPOINT_ID_HEADER,
  INVOCATION_HEADER,
  NONCE_HEADER,
  REQUEST_MAX_AGE_SECONDS,
  SIGNATURE_HEADER,
  TASK_ID_HEADER,
  TIMESTAMP_HEADER,
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

/** Parses a protojson int64 (decimal string) or a plain number into unix seconds. */
export function parseTimestamp(value: unknown): number {
  if (typeof value === 'number') {
    return Number.isFinite(value) ? value : Number.NaN;
  }

  if (typeof value === 'string' && /^-?\d+$/.test(value)) {
    return Number.parseInt(value, 10);
  }

  return Number.NaN;
}

/** Whether a signed timestamp lies within the request window of the clock, either way. */
export function isFreshTimestamp(timestampSeconds: number, nowSeconds: number): boolean {
  return (
    Number.isFinite(timestampSeconds) &&
    Math.abs(nowSeconds - timestampSeconds) <= REQUEST_MAX_AGE_SECONDS
  );
}

export interface VerifyBodyOptions {
  /** When set, bodies carrying another endpoint id are refused with 403. */
  endpointId?: string;
  /** The current time in unix seconds; defaults to the wall clock. */
  nowSeconds?: number;
}

export type BodyVerification =
  | { ok: true; json: Record<string, unknown>; timestamp: number }
  | { ok: false; status: 400 | 401 | 403; reason: string };

/**
 * Verifies a signed POST in the contract's order: the HMAC over the raw body, the body as
 * JSON, the `timestamp` it carries (within the window either way), then the `endpointId`
 * when one is configured. Returns the parsed body for the caller to decode.
 */
export async function verifySignedBody(
  body: string,
  header: string | null | undefined,
  secret: string,
  opts: VerifyBodyOptions = {}
): Promise<BodyVerification> {
  if (!(await verifyBodySignature(body, header, secret))) {
    return { ok: false, status: 401, reason: 'bad signature' };
  }

  let json: unknown;

  try {
    json = JSON.parse(body);
  } catch {
    return { ok: false, status: 400, reason: 'the body is not JSON' };
  }

  if (typeof json !== 'object' || json === null || Array.isArray(json)) {
    return { ok: false, status: 400, reason: 'the body is not a JSON object' };
  }

  const record = json as Record<string, unknown>;
  const timestamp = parseTimestamp(record.timestamp);
  const now = opts.nowSeconds ?? Math.floor(Date.now() / 1000);

  if (!isFreshTimestamp(timestamp, now)) {
    return { ok: false, status: 401, reason: 'stale or missing timestamp' };
  }

  if (opts.endpointId && record.endpointId !== opts.endpointId) {
    return { ok: false, status: 403, reason: 'unknown endpoint id' };
  }

  return { ok: true, json: record, timestamp };
}

export type UpgradeVerification =
  | { ok: true; endpointId: string; taskId: string; invocation: number; nonce: string }
  | { ok: false; status: 401 | 403 | 503; reason: string };

/** What consuming an upgrade nonce found; `full` means no room for a new one right now. */
export type NonceOutcome = 'accepted' | 'replayed' | 'full';

export interface VerifyUpgradeOptions {
  /** When set, upgrades carrying another endpoint id are refused with 403. */
  endpointId?: string;
  /** The current time in unix seconds; defaults to the wall clock. */
  nowSeconds?: number;
  /**
   * Consumes the nonce after the signature verified, so unsigned traffic cannot fill the
   * store. `replayed` refuses the upgrade with 401; `full` refuses it with 503 so the
   * operator retries later.
   */
  consumeNonce?: (nonce: string) => NonceOutcome;
}

/**
 * Verifies the bodyless websocket upgrade (pkg/serverlessoperator/durable/dial.go) in the
 * contract's order: the endpoint id header is required and, when one is configured, must be
 * this endpoint's; the timestamp must be within the window either way; the HMAC covers
 * `upgradeSigningPayload` with the header's endpoint id; then the nonce is consumed from the
 * store only once the signature holds. Exported so adapters can verify before accepting.
 */
export async function verifyUpgradeSignature(
  headers: Headers,
  secret: string,
  opts: VerifyUpgradeOptions = {}
): Promise<UpgradeVerification> {
  const endpointId = headers.get(ENDPOINT_ID_HEADER) ?? '';

  if (endpointId === '') {
    return { ok: false, status: 401, reason: 'missing endpoint id' };
  }

  if (opts.endpointId && endpointId !== opts.endpointId) {
    return { ok: false, status: 403, reason: 'unknown endpoint id' };
  }

  const timestamp = headers.get(TIMESTAMP_HEADER) ?? '';
  const now = opts.nowSeconds ?? Math.floor(Date.now() / 1000);

  if (!isFreshTimestamp(parseTimestamp(timestamp), now)) {
    return { ok: false, status: 401, reason: 'stale or missing timestamp' };
  }

  const nonce = headers.get(NONCE_HEADER) ?? '';
  const taskId = headers.get(TASK_ID_HEADER) ?? '';
  const invocation = headers.get(INVOCATION_HEADER) ?? '';
  const expected = await signHex(
    secret,
    upgradeSigningPayload(endpointId, timestamp, nonce, taskId, invocation)
  );

  if (!constantTimeEqual(expected, headers.get(SIGNATURE_HEADER) ?? '')) {
    return { ok: false, status: 401, reason: 'bad signature' };
  }

  if (nonce === '') {
    return { ok: false, status: 401, reason: 'missing or replayed nonce' };
  }

  switch (opts.consumeNonce?.(nonce) ?? 'accepted') {
    case 'replayed':
      return { ok: false, status: 401, reason: 'missing or replayed nonce' };
    case 'full':
      return { ok: false, status: 503, reason: 'nonce store full' };
    default:
      break;
  }

  return { ok: true, endpointId, taskId, invocation: Number.parseInt(invocation, 10), nonce };
}
