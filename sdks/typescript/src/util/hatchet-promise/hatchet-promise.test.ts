import HatchetPromise, { CancellationReason } from './hatchet-promise';
import {
  TaskRunTerminatedError,
  isTaskRunTerminatedError,
} from '@util/errors/task-run-terminated-error';

describe('HatchetPromise', () => {
  it('should resolve the original promise if not canceled', async () => {
    const hatchetPromise = new HatchetPromise(
      new Promise((resolve) => {
        setTimeout(() => resolve('RESOLVED'), 500);
      })
    );
    const result = await hatchetPromise.promise;
    expect(result).toEqual('RESOLVED');
  });
  it('should reject with a TaskRunTerminatedError when canceled', async () => {
    const hatchetPromise = new HatchetPromise(
      new Promise((resolve) => {
        setTimeout(() => resolve('RESOLVED'), 500);
      })
    );

    const result = hatchetPromise.promise;
    setTimeout(() => {
      hatchetPromise.cancel();
    }, 100);

    try {
      await result;
      expect(true).toEqual(false);
    } catch (e) {
      expect(isTaskRunTerminatedError(e)).toBe(true);
      expect((e as TaskRunTerminatedError).reason).toBe('cancelled');
    }
  });
  it('should use evicted reason when cancelled with EVICTED_BY_WORKER', async () => {
    const hatchetPromise = new HatchetPromise(
      new Promise((resolve) => {
        setTimeout(() => resolve('RESOLVED'), 500);
      })
    );

    const result = hatchetPromise.promise;
    setTimeout(() => {
      hatchetPromise.cancel(CancellationReason.EVICTED_BY_WORKER);
    }, 100);

    try {
      await result;
      expect(true).toEqual(false);
    } catch (e) {
      expect(isTaskRunTerminatedError(e)).toBe(true);
      expect((e as TaskRunTerminatedError).reason).toBe('evicted');
    }
  });
});
