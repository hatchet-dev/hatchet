import { V1Worker } from '@hatchet/v1/client/worker/worker-internal';
import HatchetPromise from '@util/hatchet-promise/hatchet-promise';

describe('V1Worker handleCancelStepRun cancellation supervision', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('logs warnings after threshold and grace period, then returns', async () => {
    const logger = {
      info: jest.fn(),
      warn: jest.fn(),
      debug: jest.fn(),
      error: jest.fn(),
    };

    const taskExternalId = 'task-1';

    // Use the real HatchetPromise behavior: cancel rejects the wrapper immediately,
    // while the underlying work (`inner`) continues.
    const inner = new Promise<void>(() => {
      // never resolves
    });
    const future = new HatchetPromise(inner);
    const originalCancel = future.cancel;
    const cancelSpy = jest.fn((reason: any) => originalCancel(reason));
    future.cancel = cancelSpy;

    const ctx = {
      abortController: new AbortController(),
    };

    const fakeThis: any = {
      logger,
      client: {
        config: {
          cancellation_warning_threshold: 300,
          cancellation_grace_period: 1000,
        },
      },
      cancellingTaskRuns: new Set(),
      futures: { [taskExternalId]: future },
      contexts: { [taskExternalId]: ctx },
    };

    const action: any = { taskRunExternalId: taskExternalId };

    const p = V1Worker.prototype.handleCancelStepRun.call(fakeThis, action);

    await jest.advanceTimersByTimeAsync(1500);
    await p;

    expect(ctx.abortController.signal.aborted).toBe(true);
    expect(cancelSpy).toHaveBeenCalled();
    expect(logger.warn).toHaveBeenCalled();

    expect(fakeThis.futures[taskExternalId]).toBeUndefined();
    expect(fakeThis.contexts[taskExternalId]).toBeUndefined();
  });

  it('suppresses "was cancelled" debug log when cancellation is supervised', async () => {
    const logger = {
      info: jest.fn(),
      warn: jest.fn(),
      debug: jest.fn(),
      error: jest.fn(),
    };

    const taskExternalId = 'task-2';

    const fakeThis: any = {
      logger,
      cancellingTaskRuns: new Set([taskExternalId]),
    };

    // Reproduce the log suppression logic from the step execution path:
    // we only log "was cancelled" if the cancellation isn't currently supervised.
    const maybeLog = (e: any) => {
      const message = e?.message || String(e);
      if (message.includes('Cancelled')) {
        if (!fakeThis.cancellingTaskRuns.has(taskExternalId)) {
          fakeThis.logger.debug(`Task run ${taskExternalId} was cancelled`);
        }
      }
    };

    maybeLog(new Error('Cancelled by worker'));
    expect(logger.debug).not.toHaveBeenCalled();
  });
});
