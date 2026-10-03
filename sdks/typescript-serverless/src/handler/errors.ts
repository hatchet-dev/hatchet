/**
 * Thrown by every `ctx` member the serverless runtime cannot serve as configured. Most of
 * them need a Hatchet client, which the handler builds from its `client` option; a few need
 * the invocation socket, which the operator opens for the tasks listed in `streams`; and a
 * few need a worker, which never exists here. Retrying cannot help, so the trigger handler
 * reports the failure as permanent (422, retry false).
 */
export class ServerlessLimitationError extends Error {
  /** The feature that was requested, as the user would name it, for example `ctx.runChild`. */
  readonly feature: string;

  /**
   * @param feature - The `ctx` member, as the user would name it.
   * @param detail - What to do about it, appended after the reason.
   * @param reason - Why the feature is unavailable; by default it needs the `client` option.
   */
  constructor(feature: string, detail?: string, reason = 'configure `client` to enable this') {
    super(
      `${feature} is not available in the serverless runtime: ${reason}` +
        (detail ? `. ${detail}` : '') +
        '. See LIMITATIONS.md in @hatchet-dev/serverless.'
    );
    this.name = 'ServerlessLimitationError';
    this.feature = feature;
  }
}

export function isServerlessLimitationError(err: unknown): err is ServerlessLimitationError {
  return (
    err instanceof ServerlessLimitationError || (err as Error)?.name === 'ServerlessLimitationError'
  );
}
