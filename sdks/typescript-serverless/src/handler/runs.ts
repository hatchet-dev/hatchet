/**
 * Waiting for a run's result from inside a task. The core client's `WorkflowRunRef.result()`
 * polls `GetRunDetails`, which is right outside a task; inside one the invocation socket is
 * there, so the result is awaited on one `SubscribeToWorkflowRuns` stream the task keeps
 * open over it: every child run is one more subscription on that stream, and the engine
 * sends each run's terminal event once. One stream serves every child of the task, so a
 * fan-out of a hundred children takes one of the socket's stream slots.
 *
 * A task invoked over a plain POST has no socket. Its references are still real (the id,
 * `cancel`, `replay` and `details` work), but `result()` throws a `ServerlessLimitationError`
 * naming the `streams` option that gives the task a socket.
 */
import {
  HatchetError,
  WorkflowRunRef,
  type ResultOptions,
  type RunRefClient,
} from '@hatchet-dev/typescript-sdk/core/index.js';
import { createAbortError } from '@hatchet-dev/typescript-sdk/edge/index.js';
import {
  SubscribeToWorkflowRunsRequest,
  WorkflowRunEvent,
  WorkflowRunEventType,
} from '../generated/proto/dispatcher';
import {
  SUBSCRIBE_TO_WORKFLOW_RUNS,
  type SocketStream,
  type SocketStreams,
} from './durable/streams';
import { ServerlessLimitationError } from './errors';

/** The stream that carried a run's subscription ended before the run's terminal event. */
export class RunStreamClosedError extends Error {
  constructor(
    readonly workflowRunId: string,
    readonly code: number,
    detail: string
  ) {
    super(
      `the run stream closed before run ${workflowRunId} finished (code ${code}${detail ? `: ${detail}` : ''})`
    );
    this.name = 'RunStreamClosedError';
  }
}

interface Waiter {
  resolve: (event: WorkflowRunEvent) => void;
  reject: (reason: unknown) => void;
}

/**
 * Awaits runs' terminal events on one `SubscribeToWorkflowRuns` stream over the socket. The
 * stream is opened on the first wait and reopened after the operator closes it, as long as
 * the socket is up; every waiter of a run that was awaited on a stream that ended is rejected
 * with `RunStreamClosedError`.
 */
export class RunWatcher {
  private stream: SocketStream | undefined;
  private readonly waiters = new Map<string, Set<Waiter>>();

  constructor(private readonly streams: SocketStreams) {}

  /** Whether the runs stream is open right now. */
  get subscribed(): boolean {
    return this.stream !== undefined;
  }

  /**
   * Resolves with the run's terminal event. `signal` rejects with an `AbortError`,
   * `timeoutMs` with a `HatchetError`, the same as the core reference's `result()`.
   */
  awaitRun(workflowRunId: string, options: ResultOptions = {}): Promise<WorkflowRunEvent> {
    const { signal, timeoutMs } = options;

    if (signal?.aborted) {
      return Promise.reject(createAbortError(`waiting for run ${workflowRunId} was aborted`));
    }

    return new Promise<WorkflowRunEvent>((resolve, reject) => {
      let settled = false;
      let timer: ReturnType<typeof setTimeout> | undefined;

      const waiter: Waiter = {
        resolve: (event) => finish(() => resolve(event)),
        reject: (reason) => finish(() => reject(reason)),
      };

      const onAbort = () =>
        waiter.reject(createAbortError(`waiting for run ${workflowRunId} was aborted`));

      const finish = (settle: () => void) => {
        if (settled) {
          return;
        }

        settled = true;
        signal?.removeEventListener('abort', onAbort);

        if (timer !== undefined) {
          clearTimeout(timer);
        }

        this.forget(workflowRunId, waiter);
        settle();
      };

      const set = this.waiters.get(workflowRunId) ?? new Set<Waiter>();
      const first = set.size === 0;
      set.add(waiter);
      this.waiters.set(workflowRunId, set);

      signal?.addEventListener('abort', onAbort, { once: true });

      if (timeoutMs !== undefined) {
        timer = setTimeout(
          () =>
            waiter.reject(
              new HatchetError(`timed out after ${timeoutMs} ms waiting for run ${workflowRunId}`)
            ),
          timeoutMs
        );
      }

      // A run already awaited is subscribed already; the engine sends its event once.
      if (first) {
        try {
          this.subscribe(workflowRunId);
        } catch (err) {
          waiter.reject(err);
        }
      }
    });
  }

  private forget(workflowRunId: string, waiter: Waiter): void {
    const set = this.waiters.get(workflowRunId);

    if (set?.delete(waiter) && set.size === 0) {
      this.waiters.delete(workflowRunId);
    }
  }

  private subscribe(workflowRunId: string): void {
    const request = JSON.stringify(
      SubscribeToWorkflowRunsRequest.toJSON(
        SubscribeToWorkflowRunsRequest.create({ workflowRunId })
      )
    );

    if (this.stream) {
      this.stream.send(request);
      return;
    }

    this.stream = this.streams.openStream(SUBSCRIBE_TO_WORKFLOW_RUNS, request, {
      onMessage: (message) => this.onEvent(message),
      onClose: (code, detail) => this.onClosed(code, detail),
    });
  }

  private onEvent(message: string): void {
    let event: WorkflowRunEvent;

    try {
      event = WorkflowRunEvent.fromJSON(JSON.parse(message));
    } catch {
      return;
    }

    if (event.eventType !== WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_FINISHED) {
      return;
    }

    const set = this.waiters.get(event.workflowRunId);

    if (!set) {
      return;
    }

    for (const waiter of [...set]) {
      waiter.resolve(event);
    }
  }

  private onClosed(code: number, detail: string): void {
    this.stream = undefined;

    for (const [workflowRunId, set] of [...this.waiters]) {
      for (const waiter of [...set]) {
        waiter.reject(new RunStreamClosedError(workflowRunId, code, detail));
      }
    }
  }

  /** Closes the runs stream; pending waits keep waiting only until the socket ends. */
  close(): void {
    this.stream?.close();
    this.stream = undefined;
  }
}

/**
 * The outputs a terminal event carries, read the way the Node client's `result()` reads
 * them: each task's output parsed as JSON (`{}` when empty), keyed by task name, or the
 * standalone task's own output.
 */
export function outputsOf<T>(event: WorkflowRunEvent, standaloneTaskName?: string): T {
  const outputs: Record<string, unknown> = {};

  for (const result of event.results) {
    outputs[result.taskName] = JSON.parse(result.output || '{}');
  }

  if (standaloneTaskName) {
    return outputs[standaloneTaskName] as T;
  }

  return outputs as T;
}

/**
 * A reference to a run triggered from inside a task. `result()` awaits the run's terminal
 * event on the invocation socket; without a socket it throws, since a POST-invoked task has
 * nothing to await on. Everything else is the core client's reference.
 */
export class TaskRunRef<T> extends WorkflowRunRef<T> {
  constructor(
    workflowRunId: string,
    runs: RunRefClient,
    private readonly watcher: RunWatcher | undefined,
    parentWorkflowRunId?: string,
    standaloneTaskName?: string,
    defaultSignal?: AbortSignal
  ) {
    super(workflowRunId, runs, parentWorkflowRunId, standaloneTaskName, defaultSignal);
  }

  /**
   * Waits for the run to finish. A run whose tasks reported errors rejects with the array
   * of their messages, as the Node client's `result()` does; a terminal event that carries no
   * task results (a run that failed before any task was dispatched) is read from
   * `GetRunDetails` instead, which is one unary call since the run is over.
   */
  override async result(options: ResultOptions = {}): Promise<T> {
    if (!this.watcher) {
      throw new ServerlessLimitationError(
        'awaiting a child run (ctx.runChild, ctx.bulkRunChildren, ref.result())',
        "List the task in the handler's `streams` option so the operator invokes it over the invocation socket, where results are awaited",
        'this task was invoked over a POST and has no invocation socket'
      );
    }

    const signal = options.signal ?? this.defaultSignal;
    const event = await this.watcher.awaitRun(this.workflowRunId, { ...options, signal });
    const errors = event.results
      .map((result) => result.error)
      .filter((error): error is string => error !== undefined && error !== '');

    if (errors.length > 0) {
      throw errors;
    }

    if (event.results.length === 0) {
      return super.result(options);
    }

    return outputsOf<T>(event, this._standaloneTaskName);
  }
}
