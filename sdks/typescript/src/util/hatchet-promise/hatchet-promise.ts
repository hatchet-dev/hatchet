import { TaskRunTerminatedError } from '@util/errors/task-run-terminated-error';

/** Canonical reasons when cancelling a HatchetPromise (e.g. worker shutdown). */
export enum CancellationReason {
  CANCELLED_BY_WORKER = 'Cancelled by worker',
  EVICTED_BY_WORKER = 'Evicted by worker',
}

const reasonToTermination: Record<CancellationReason, 'cancelled' | 'evicted'> = {
  [CancellationReason.CANCELLED_BY_WORKER]: 'cancelled',
  [CancellationReason.EVICTED_BY_WORKER]: 'evicted',
};

class HatchetPromise<T> {
  cancel: (reason?: CancellationReason) => void = (_reason?: CancellationReason) => {};
  promise: Promise<T>;
  /**
   * The original (non-cancelable) promise passed to the constructor.
   *
   * `promise` is a cancelable wrapper which rejects immediately when `cancel` is called.
   * `inner` continues executing and will settle when the underlying work completes.
   */
  inner: Promise<T>;

  constructor(promise: Promise<T>) {
    this.inner = Promise.resolve(promise) as Promise<T>;
    this.promise = new Promise((resolve, reject) => {
      this.cancel = (reason?: CancellationReason) => {
        const termination = reason ? reasonToTermination[reason] : 'cancelled';
        reject(new TaskRunTerminatedError(termination, reason));
      };
      this.inner.then(resolve).catch(reject);
    });
  }
}

export default HatchetPromise;
