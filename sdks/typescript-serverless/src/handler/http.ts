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
