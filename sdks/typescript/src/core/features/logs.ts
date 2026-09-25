import { LogLevel } from '@hatchet/clients/event/rpc';
import type { EventsServiceClient, PutLogResponse } from '@hatchet/protoc/events/events';
import HatchetError from '@util/errors/hatchet-error';

/** The longest log line accepted, in characters; the Node client's `putLog` holds the same limit. */
export const MAX_LOG_LINE_CHARS = 1_000;

export interface PutLogOptions {
  /** Defaults to `INFO`. */
  level?: LogLevel;
  /** The attempt the line belongs to, when the task has retried. */
  retryCount?: number;
  metadata?: Record<string, unknown>;
}

/** Writes log lines against a task run, the way `ctx.log` does from a worker. */
export class LogsClient {
  constructor(private readonly rpc: EventsServiceClient) {}

  /**
   * Stores one log line for a task run. Unlike the Node client's fire-and-forget `putLog`,
   * this resolves once the engine has the line and rejects when it does not, since a
   * serverless runtime may end the invocation as soon as the handler returns.
   *
   * A line over `MAX_LOG_LINE_CHARS` (1,000) characters rejects with a `HatchetError` that
   * states the length, where the Node client warns and drops the line; the message itself is
   * never part of the error.
   */
  async put(
    taskRunExternalId: string,
    message: string,
    options: PutLogOptions = {}
  ): Promise<PutLogResponse> {
    if (message.length > MAX_LOG_LINE_CHARS) {
      throw new HatchetError(
        `log line is ${message.length} characters, over the ${MAX_LOG_LINE_CHARS}-character limit`
      );
    }

    return this.rpc.putLog({
      taskRunExternalId,
      createdAt: new Date(),
      message,
      level: options.level ?? LogLevel.INFO,
      taskRetryCount: options.retryCount,
      metadata: options.metadata ? JSON.stringify(options.metadata) : undefined,
    });
  }
}
