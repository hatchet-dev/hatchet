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

/**
 * `@grpc/grpc-js` falls back to mapping a plain HTTP status code onto a gRPC
 * status (e.g. 404 -> UNIMPLEMENTED, 502 -> UNAVAILABLE, other -> UNKNOWN)
 * whenever it receives a non-gRPC-framed HTTP response - typically because a
 * proxy/load balancer served the response instead of the real gRPC server
 * (e.g. while the engine is restarting behind it). It stamps these with a
 * distinctive details string, which lets us tell "the transport made this up"
 * apart from a genuine status returned by the application itself.
 */
export function isHttpMappedStatus(e: unknown): boolean {
  const details = getGrpcErrorDetails(e);
  return details !== undefined && details.startsWith('Received HTTP status code');
}

/**
 * True for errors that most likely represent the server being transiently
 * unreachable (a real connection-level gRPC code, or a status that grpc-js
 * fabricated from a raw HTTP response) rather than a genuine application error.
 */
export function isTransientConnectionIssue(e: unknown): boolean {
  return isConnectionError(getGrpcErrorCode(e)) || isHttpMappedStatus(e);
}
