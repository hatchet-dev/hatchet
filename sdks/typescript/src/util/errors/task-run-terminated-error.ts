export type TaskRunTerminationReason = 'cancelled' | 'evicted';

export class TaskRunTerminatedError extends Error {
  readonly reason: TaskRunTerminationReason;

  constructor(reason: TaskRunTerminationReason, message?: string) {
    super(message ?? reason);
    this.name = 'TaskRunTerminatedError';
    this.reason = reason;
  }
}

export function isTaskRunTerminatedError(err: unknown): err is TaskRunTerminatedError {
  return err instanceof TaskRunTerminatedError;
}
