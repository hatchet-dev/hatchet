import { LogLevel } from '@hatchet/clients/event/rpc';
import type { EventsServiceClient, PutLogResponse } from '@hatchet/protoc/events/events';

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
   */
  put(
    taskRunExternalId: string,
    message: string,
    options: PutLogOptions = {}
  ): Promise<PutLogResponse> {
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
