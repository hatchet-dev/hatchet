import HatchetError from '@util/errors/hatchet-error';
import { createAbortError } from '@hatchet/util/abort-error';
import { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import type { CallOptions, ResultOptions, RunDetail } from './types';

/** What a run reference needs from the client that created it. */
export interface RunRefClient {
  getDetails(runId: string, options?: CallOptions): Promise<RunDetail>;
  cancel(opts: { ids: string[] }): Promise<unknown>;
  replay(opts: { ids: string[] }): Promise<unknown>;
}

/** The first wait between two `GetRunDetails` polls. */
export const INITIAL_POLL_INTERVAL_MS = 250;
/** The longest wait between two polls; the interval doubles until it gets here. */
export const MAX_POLL_INTERVAL_MS = 5_000;

/**
 * The signal one poll is issued with: it fires when the caller's signal fires or when the
 * deadline arrives, whichever is first. `AbortSignal.any` and `AbortSignal.timeout` are
 * missing on some runtimes the core entry targets, so a plain controller, timer and
 * forwarded abort are used everywhere; the timer is also one a test's fake clock controls.
 */
function pollSignal(
  signal: AbortSignal | undefined,
  remainingMs: number | undefined
): { signal: AbortSignal; timedOut: () => boolean; dispose: () => void } {
  const controller = new AbortController();
  let timedOut = false;

  const forwardAbort = () => controller.abort(signal?.reason);
  signal?.addEventListener('abort', forwardAbort, { once: true });

  const timer =
    remainingMs === undefined
      ? undefined
      : setTimeout(() => {
          timedOut = true;
          controller.abort();
        }, remainingMs);

  return {
    signal: controller.signal,
    timedOut: () => timedOut,
    dispose: () => {
      signal?.removeEventListener('abort', forwardAbort);
      if (timer !== undefined) clearTimeout(timer);
    },
  };
}

function wait(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    if (signal?.aborted) {
      resolve();
      return;
    }
    const timer = setTimeout(() => {
      signal?.removeEventListener('abort', onAbort);
      resolve();
    }, ms);
    const onAbort = () => {
      clearTimeout(timer);
      resolve();
    };
    signal?.addEventListener('abort', onAbort, { once: true });
  });
}

/**
 * A reference to a triggered run. `result()` polls `GetRunDetails` until the run is done,
 * which is the one way to wait for a run that works on every platform the core client runs
 * on: no stream is held open, so a Worker request or a browser tab can wait on it.
 */
export class WorkflowRunRef<T> {
  readonly workflowRunId: string;
  parentWorkflowRunId?: string;
  _standaloneTaskName?: string;
  /**
   * The signal `result()` waits with when none is passed, so a run spawned from inside a task
   * stops being awaited when the task is cancelled.
   */
  defaultSignal?: AbortSignal;

  constructor(
    workflowRunId: string,
    private readonly runs: RunRefClient,
    parentWorkflowRunId?: string,
    standaloneTaskName?: string,
    defaultSignal?: AbortSignal
  ) {
    this.workflowRunId = workflowRunId;
    this.parentWorkflowRunId = parentWorkflowRunId;
    this._standaloneTaskName = standaloneTaskName;
    this.defaultSignal = defaultSignal;
  }

  get runId(): Promise<string> {
    return this.getWorkflowRunId();
  }

  async getWorkflowRunId(): Promise<string> {
    return this.workflowRunId;
  }

  /** The run's result; the same as `result()` with no options. */
  get output(): Promise<T> {
    return this.result();
  }

  /** The run's current state. */
  details(): Promise<RunDetail> {
    return this.runs.getDetails(this.workflowRunId);
  }

  /**
   * Waits for the run to finish and returns its output: the standalone task's output for a
   * task, or the outputs keyed by task name for a workflow.
   *
   * The wait polls with backoff, starting at 250 ms and capped at 5 s with jitter. It is
   * unbounded unless `timeoutMs` is set (rejects with a `HatchetError`) or `signal` fires
   * (rejects with an `AbortError`). Both bound the poll in flight as well as the waits
   * between polls, so a stalled `GetRunDetails` cannot hold the result past the deadline.
   *
   * A failed or cancelled run rejects the way the Node client's `result()` does: with the
   * array of the tasks' error messages when any task reported one, otherwise with an `Error`
   * naming the run's status.
   */
  async result(options: ResultOptions = {}): Promise<T> {
    const signal = options.signal ?? this.defaultSignal;
    const startedAt = Date.now();
    const deadline = options.timeoutMs !== undefined ? startedAt + options.timeoutMs : undefined;
    let interval = INITIAL_POLL_INTERVAL_MS;

    const abortError = () => createAbortError(`waiting for run ${this.workflowRunId} was aborted`);
    const timeoutError = () =>
      new HatchetError(
        `timed out after ${options.timeoutMs} ms waiting for run ${this.workflowRunId}`
      );

    for (;;) {
      if (signal?.aborted) {
        throw abortError();
      }
      const remaining = deadline !== undefined ? deadline - Date.now() : undefined;
      if (remaining !== undefined && remaining <= 0) {
        throw timeoutError();
      }

      // Whatever the transport rejected with, an abort by the caller's signal surfaces as the
      // AbortError the check above throws and one by the deadline as the same HatchetError;
      // the deadline also goes to the call so the transport applies a Connect timeout.
      const poll = pollSignal(signal, remaining);
      let detail: RunDetail;
      try {
        detail = await this.runs.getDetails(this.workflowRunId, { signal: poll.signal, deadline });
      } catch (e) {
        if (signal?.aborted) {
          throw abortError();
        }
        if (poll.timedOut() || (deadline !== undefined && Date.now() >= deadline)) {
          throw timeoutError();
        }
        throw e;
      } finally {
        poll.dispose();
      }
      if (detail.done) {
        return this.resolveResult(detail);
      }

      const now = Date.now();
      if (deadline !== undefined && now >= deadline) {
        throw timeoutError();
      }

      // Jitter of up to 20% keeps many waiters from polling in lockstep.
      let delay = Math.round(interval * (1 + Math.random() * 0.2));
      if (deadline !== undefined) {
        delay = Math.min(delay, Math.max(0, deadline - now));
      }
      await wait(delay, signal);
      interval = Math.min(interval * 2, MAX_POLL_INTERVAL_MS);
    }
  }

  private resolveResult(detail: RunDetail): Promise<T> {
    if (detail.status === V1TaskStatus.COMPLETED) {
      const outputs: Record<string, unknown> = {};
      for (const task of Object.values(detail.taskRuns)) {
        outputs[task.readableId] = task.output ?? {};
      }
      if (this._standaloneTaskName) {
        return Promise.resolve(outputs[this._standaloneTaskName] as T);
      }
      return Promise.resolve(outputs as T);
    }

    const errors = Object.values(detail.taskRuns)
      .map((task) => task.error)
      .filter((error): error is string => error !== undefined && error !== '');

    if (errors.length > 0) {
      return Promise.reject(errors);
    }

    const outcome = detail.status === V1TaskStatus.CANCELLED ? 'was cancelled' : 'failed';
    return Promise.reject(new Error(`run ${this.workflowRunId} ${outcome}`));
  }

  async toJSON(): Promise<string> {
    return JSON.stringify({ workflowRunId: this.workflowRunId });
  }

  async cancel() {
    await this.runs.cancel({ ids: [this.workflowRunId] });
  }

  async replay() {
    await this.runs.replay({ ids: [this.workflowRunId] });
  }
}
