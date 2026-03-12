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
