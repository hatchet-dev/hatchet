import type { DispatcherClient } from '@hatchet/protoc/dispatcher';
import { durationToString, type Duration } from '@hatchet/v1/client/duration';
import type { CallOptions } from '../types';

/**
 * Calls a running task makes about itself, over the `Dispatcher` service's unary RPCs: the
 * same calls the Node worker context exposes as `ctx.refreshTimeout()` and
 * `ctx.releaseSlot()`, for code that holds a task run's external id but runs outside a
 * worker.
 */
export class TasksClient {
  constructor(private readonly rpc: DispatcherClient) {}

  /**
   * Extends a task run's execution timeout.
   * @param taskRunExternalId - The task run's external id.
   * @param incrementBy - How much longer the task may run: a Go duration string (`'10s'`,
   * `'1m30s'`), a `{ hours, minutes, seconds }` object or a number of milliseconds.
   * @param options - A signal or deadline for the call.
   * @returns the task run's new timeout, when the engine reports it
   */
  async refreshTimeout(
    taskRunExternalId: string,
    incrementBy: Duration,
    options?: CallOptions
  ): Promise<Date | undefined> {
    const response = await this.rpc.refreshTimeout(
      { taskRunExternalId, incrementTimeoutBy: durationToString(incrementBy) },
      options
    );
    return response.timeoutAt;
  }

  /**
   * Releases the worker slot a task run holds, so the worker can pick up another task while
   * this one keeps running. An advanced call: the task is no longer counted against the
   * worker's slots, so the worker can be oversubscribed.
   * @param taskRunExternalId - The task run's external id.
   * @param options - A signal or deadline for the call.
   */
  async releaseSlot(taskRunExternalId: string, options?: CallOptions): Promise<void> {
    await this.rpc.releaseSlot({ taskRunExternalId }, options);
  }
}
