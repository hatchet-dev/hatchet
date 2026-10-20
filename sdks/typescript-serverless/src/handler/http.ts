import { ServerlessTriggerError } from '../generated/proto/v1/serverless';

const JSON_HEADERS = { 'content-type': 'application/json' };

export function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: JSON_HEADERS });
}

export function jsonText(text: string, status = 200): Response {
  return new Response(text, { status, headers: JSON_HEADERS });
}

/**
 * A non-2xx trigger response carrying the ServerlessTriggerError body, which overrides the
 * operator's default message and retry decision for the status.
 */
export function triggerError(status: number, error: string, retry: boolean): Response {
  return json(ServerlessTriggerError.toJSON({ error, retry }), status);
}

export function errorMessage(err: unknown): string {
  if (err instanceof Error) {
    return err.message;
  }

  return typeof err === 'string' ? err : String(err);
}

/** The 413 a request over `maxBodyBytes` is answered with; the operator does not retry it. */
export function bodyTooLarge(maxBytes: number): Response {
  return triggerError(413, `the request body exceeds the ${maxBytes} byte limit`, false);
}

export type BodyRead = { ok: true; text: string } | { ok: false; response: Response };

/**
 * Reads the body as UTF-8 text, giving up as soon as it is known to exceed `maxBytes`: a
 * larger `content-length` is refused without reading, and a stream is cancelled at the
 * first chunk past the cap. The signature covers these exact bytes, so nothing is parsed
 * here.
 */
export async function readBodyText(request: Request, maxBytes: number): Promise<BodyRead> {
  const declared = Number(request.headers.get('content-length'));

  if (Number.isFinite(declared) && declared > maxBytes) {
    return { ok: false, response: bodyTooLarge(maxBytes) };
  }

  if (!request.body) {
    return { ok: true, text: '' };
  }

  const reader = request.body.getReader();
  const chunks: Uint8Array[] = [];
  let received = 0;

  for (;;) {
    const { done, value } = await reader.read();

    if (done) {
      break;
    }

    received += value.byteLength;

    if (received > maxBytes) {
      await reader.cancel().catch(() => {});
      return { ok: false, response: bodyTooLarge(maxBytes) };
    }

    chunks.push(value);
  }

  const bytes = new Uint8Array(received);
  let offset = 0;

  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }

  return { ok: true, text: new TextDecoder().decode(bytes) };
}
