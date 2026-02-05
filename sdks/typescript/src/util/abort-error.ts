export function createAbortError(message = 'Operation aborted'): Error {
  const err: any = new Error(message);
  err.name = 'AbortError';
  err.code = 'ABORT_ERR';
  return err as Error;
}

export function isAbortError(err: unknown): err is Error {
  return (
    err instanceof Error &&
    (err.name === 'AbortError' || (err as any).code === 'ABORT_ERR')
  );
}

/**
 * Helper to be used inside broad `catch` blocks so cancellation isn't accidentally swallowed.
 *
 * Example:
 * ```ts
 * try { ... } catch (e) { rethrowIfAborted(e); ... }
 * ```
 */
export function rethrowIfAborted(err: unknown): void {
  if (isAbortError(err)) {
    throw err;
  }
}

export type ThrowIfAbortedOpts = {
  /**
   * Optional: called before throwing when the signal is aborted.
   * This lets callsites attach logging without coupling this util to a logger implementation.
   */
  warn?: (message: string) => void;

  /**
   * If true, emits a generic warning intended for "trigger/enqueue" paths.
   */
  isTrigger?: boolean;

  /**
   * Optional context used to make warnings consistent, e.g. "task run <id>".
   */
  context?: string;

  /**
   * Message used when the AbortSignal doesn't provide a reason.
   */
  defaultMessage?: string;
};

/**
 * Throws an AbortError if the provided signal is aborted.
 *
 * Notes:
 * - In JS/TS, `catch` can swallow any thrown value, so this is best-effort.
 * - We prefer throwing the signal's `reason` when it is already an Error.
 */
export function throwIfAborted(
  signal: AbortSignal | undefined,
  optsOrDefaultMessage: ThrowIfAbortedOpts | string = 'Operation cancelled by AbortSignal'
): void {
  if (!signal?.aborted) {
    return;
  }

  const opts: ThrowIfAbortedOpts =
    typeof optsOrDefaultMessage === 'string'
      ? { defaultMessage: optsOrDefaultMessage }
      : (optsOrDefaultMessage ?? {});

  if (opts.isTrigger) {
    const ctx = opts.context ? `${opts.context} ` : '';
    opts.warn?.(
      `Cancellation: ${ctx}attempted to enqueue/trigger work after cancellation was signaled. ` +
        `This usually means an AbortError was caught and not propagated. ` +
        `See https://docs.hatchet.run/home/cancellation`
    );
  }

  const reason = (signal as any).reason;

  if (reason instanceof Error) {
    throw reason;
  }

  if (typeof reason === 'string' && reason.length > 0) {
    throw createAbortError(reason);
  }

  throw createAbortError(opts.defaultMessage ?? 'Operation cancelled by AbortSignal');
}
