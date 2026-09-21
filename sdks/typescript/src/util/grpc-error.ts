import { Status } from 'nice-grpc';

/**
 * gRPC codes that typically indicate a transient connectivity problem
 * (server unreachable/restarting) rather than an application-level error.
 */
export const CONNECTION_ERROR_CODES: number[] = [Status.UNAVAILABLE, Status.FAILED_PRECONDITION];

export function isConnectionError(code: number | undefined): boolean {
  return code !== undefined && CONNECTION_ERROR_CODES.includes(code);
}

/**
 * Returns true if `e` is the DOMException thrown when an AbortSignal (e.g. from
 * AbortSignal.timeout()) fires. This rejects the underlying call directly rather than
 * surfacing as a gRPC status code, so it needs to be classified as transient separately
 * from isConnectionError.
 */
export function isTimeoutOrAbortError(e: unknown): boolean {
  return e instanceof DOMException && (e.name === 'TimeoutError' || e.name === 'AbortError');
}

/**
 * Returns the gRPC status code from an unknown value (e.g. from a catch block).
 * Used for checking Status.CANCELLED, Status.UNAVAILABLE, etc.
 */
export function getGrpcErrorCode(e: unknown): number | undefined {
  if (e != null && typeof e === 'object' && 'code' in e) {
    const { code } = e as { code: unknown };
    return typeof code === 'number' ? code : undefined;
  }
  return undefined;
}

/**
 * Returns the gRPC error details string from an unknown value (e.g. from a catch block).
 */
export function getGrpcErrorDetails(e: unknown): string | undefined {
  if (e != null && typeof e === 'object' && 'details' in e) {
    const { details } = e as { details: unknown };
    return typeof details === 'string' ? details : undefined;
  }
  return undefined;
}
