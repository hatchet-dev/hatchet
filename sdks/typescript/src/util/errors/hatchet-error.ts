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

export function toHatchetError(
  e: unknown,
  defaultMessageOrOptions:
    | string
    | { defaultMessage?: string; prefix?: string } = 'An error occurred'
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
  if (opts.prefix) {
    message = opts.prefix + message;
  }
  return new HatchetError(message, { cause: e });
}

export default HatchetError;
