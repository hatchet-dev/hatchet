/**
 * Thrown by every `ctx` member that would need a Hatchet client. No client exists in the
 * serverless runtime: most serverless platforms cannot speak gRPC, and the transport that
 * will replace it is not decided. Retrying cannot help, so the trigger handler reports it
 * as a permanent failure (422, retry false).
 */
export class ServerlessLimitationError extends Error {
  /** The feature that was requested, as the user would name it, for example `ctx.runChild`. */
  readonly feature: string;

  constructor(feature: string, detail?: string) {
    super(
      `${feature} is not available in the serverless runtime: no Hatchet client exists there` +
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
