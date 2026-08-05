/** Returns a string message from an unknown value (e.g. from a catch block). */
export function getErrorMessage(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

class HatchetError extends Error {
  constructor(message: string, options?: { cause?: unknown }) {
    super(message, options);
    this.name = 'HatchetError';
  }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} bytes`;
  }
  const units = ['KB', 'MB', 'GB'];
  let value = bytes;
  let unitIndex = -1;
  do {
    value /= 1024;
    unitIndex += 1;
  } while (value >= 1024 && unitIndex < units.length - 1);
  return `${value.toFixed(2)} ${units[unitIndex]} (${bytes.toLocaleString()} bytes)`;
}

/**
 * Returns true if `e` is a gRPC RESOURCE_EXHAUSTED error caused by an oversized outgoing
 * message (as opposed to some other RESOURCE_EXHAUSTED condition, e.g. a tenant quota limit).
 */
function isMessageTooLargeError(e: unknown): boolean {
  if (e == null || typeof e !== 'object' || !('code' in e)) {
    return false;
  }
  const { code } = e as { code: unknown };
  // nice-grpc-common Status.RESOURCE_EXHAUSTED === 8
  if (code !== 8) {
    return false;
  }
  const details = 'details' in e ? String((e as { details: unknown }).details ?? '') : '';
  const lowered = details.toLowerCase();
  return lowered.includes('larger than') || lowered.includes('too large');
}

export function toHatchetError(
  e: unknown,
  defaultMessageOrOptions:
    string | { defaultMessage?: string; prefix?: string } = 'An error occurred',
  payloadSizeBytes?: number
): HatchetError {
  if (e instanceof HatchetError) {
    return e;
  }
  const opts =
    typeof defaultMessageOrOptions === 'string'
      ? { defaultMessage: defaultMessageOrOptions }
      : defaultMessageOrOptions;
  const defaultMessage = opts.defaultMessage ?? 'An error occurred';
  let message = getErrorMessage(e) || defaultMessage;

  if (payloadSizeBytes !== undefined && isMessageTooLargeError(e)) {
    message =
      `Payload too large: attempted to send ${formatBytes(payloadSizeBytes)}, which exceeds ` +
      `the gRPC max message size configured for this client.` +
      `(${message})`;
  }

  if (opts.prefix) {
    message = opts.prefix + message;
  }
  return new HatchetError(message, { cause: e });
}

export default HatchetError;
