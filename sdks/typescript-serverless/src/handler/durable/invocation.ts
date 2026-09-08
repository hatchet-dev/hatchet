/**
 * One durable invocation over an accepted websocket: wait for the operator's first frame,
 * run the durable task under a `DurableContext` whose transport is the socket, and finish
 * with exactly one done frame.
 *
 * Outcomes, in the order the operator checks them: `done { status: "evicted" }` after an
 * eviction ack (the engine re-invokes the task when the awaited entry is satisfied),
 * `done { error, retry }` on failure, `done { output }` on success. After an operator
 * `error` frame the outcome is failed without retry and the endpoint still sends the done
 * frame; after a `serverEvict` the operator closes the socket itself and nothing more is
 * sent. The socket is closed with 1000 after the done frame.
 */
import {
  DurableContext,
  MinEngineVersion,
  NonRetryableError,
  TaskRunTerminatedError,
  parentRunContextManager,
} from '@hatchet-dev/typescript-sdk/edge/index.js';
import type {
  ServerlessDurableFrame,
  ServerlessFirstFrame,
} from '../../generated/proto/v1/serverless';
import { toSdkAction } from '../action';
import { DONE_STATUS_EVICTED, stripNamespace } from '../contract';
import { ServerlessRuntime, type ConsoleLike } from '../context';
import { errorMessage } from '../http';
import type { Registry } from '../registry';
import { decodeFrame } from './frames';
import type { DurableSocket } from './socket';
import { FrameTransport } from './transport';

export interface DurableInvocationOptions {
  socket: DurableSocket;
  registry: Registry;
  console?: ConsoleLike;
  /** How long to wait for the operator's first frame after the upgrade. Defaults to 30 s. */
  firstFrameTimeoutMs?: number;
  /**
   * The engine version the context believes it talks to. The operator relays to an engine
   * with durable eviction, so the minimum version that enables it is assumed.
   */
  engineVersion?: string;
  /**
   * The task id and invocation the upgrade headers were signed for. A first frame naming
   * another task or invocation closes the socket with 1008 before any task code runs.
   */
  expected?: { taskRunExternalId: string; invocationCount: number };
}

type State =
  'awaiting-first' | 'running' | 'evicting' | 'evicted' | 'engine-error' | 'closed' | 'finished';

const DEFAULT_FIRST_FRAME_TIMEOUT_MS = 30_000;

/** Runs one durable invocation on the socket; resolves once the invocation is over. */
export function runDurableInvocation(options: DurableInvocationOptions): Promise<void> {
  return new DurableInvocation(options).run();
}

class DurableInvocation {
  private state: State = 'awaiting-first';
  private transport: FrameTransport | undefined;
  private ctx: DurableContext<unknown, unknown> | undefined;
  private engineError: Error | undefined;
  private firstFrame: ((frame: ServerlessFirstFrame) => void) | undefined;
  private firstFrameTimer: ReturnType<typeof setTimeout> | undefined;
  private readonly out: ConsoleLike;

  constructor(private readonly options: DurableInvocationOptions) {
    this.out = options.console ?? console;
  }

  async run(): Promise<void> {
    const { socket } = this.options;

    socket.onMessage((text) => this.onMessage(text));
    socket.onClose((code, reason) => this.onClose(code, reason));

    const first = await this.awaitFirstFrame();

    if (!first) {
      return;
    }

    await this.execute(first);
  }

  private awaitFirstFrame(): Promise<ServerlessFirstFrame | undefined> {
    return new Promise((resolve) => {
      const timeoutMs = this.options.firstFrameTimeoutMs ?? DEFAULT_FIRST_FRAME_TIMEOUT_MS;

      this.firstFrame = (frame) => {
        this.clearFirstFrameTimer();
        this.firstFrame = undefined;
        resolve(frame);
      };

      this.firstFrameTimer = setTimeout(() => {
        this.firstFrameTimer = undefined;
        this.firstFrame = undefined;
        this.finish({ error: `no first frame within ${timeoutMs}ms of the upgrade`, retry: true });
        resolve(undefined);
      }, timeoutMs);
    });
  }

  private clearFirstFrameTimer(): void {
    if (this.firstFrameTimer !== undefined) {
      clearTimeout(this.firstFrameTimer);
      this.firstFrameTimer = undefined;
    }
  }

  private onMessage(text: string): void {
    let frame: ServerlessDurableFrame;

    try {
      frame = decodeFrame(text);
    } catch (err) {
      this.out.warn(`[hatchet] malformed durable frame from the operator: ${errorMessage(err)}`);
      return;
    }

    if (this.state === 'awaiting-first') {
      if (frame.first && this.firstFrame) {
        this.firstFrame(frame.first);
      }

      return;
    }

    this.transport?.handleFrame(frame);
  }

  private async execute(first: ServerlessFirstFrame): Promise<void> {
    const { registry } = this.options;

    if (!first.action) {
      this.finish({ error: 'the first frame carries no action', retry: false });
      return;
    }

    const { expected } = this.options;

    if (
      expected &&
      (first.action.taskRunExternalId !== expected.taskRunExternalId ||
        first.action.durableTaskInvocationCount !== expected.invocationCount ||
        first.invocationCount !== expected.invocationCount)
    ) {
      this.out.warn(
        `[hatchet] durable first frame names task ${first.action.taskRunExternalId} invocation ${first.invocationCount}, but the upgrade was signed for task ${expected.taskRunExternalId} invocation ${expected.invocationCount}; closing`
      );
      this.state = 'closed';
      this.closeWith(1008, 'first frame does not match the signed upgrade');
      return;
    }

    const actionId = stripNamespace(first.action.actionId, first.namespace);
    const runner = registry.durableRunners.get(actionId);

    if (!runner) {
      const error = registry.runners.has(actionId)
        ? `action ${actionId} is not a durable task; it is served over POST, not the websocket relay`
        : `no durable task served for action ${actionId}`;

      this.finish({ error, retry: false });
      return;
    }

    const action = toSdkAction(first.action, first.namespace);
    action.durableTaskInvocationCount = first.invocationCount;

    const transport = new FrameTransport({
      socket: this.options.socket,
      durableTaskExternalId: action.taskRunExternalId,
      invocationCount: first.invocationCount,
      inlineWaitBudgetMs: first.inlineWaitBudgetMs,
      onBudgetElapsed: (reason) => void this.evict(reason),
      onEngineError: (error) => this.onEngineError(error),
      onServerEvict: (reason) => this.onServerEvict(reason),
    });

    let ctx: DurableContext<unknown, unknown>;

    try {
      ctx = new DurableContext(
        action,
        new ServerlessRuntime({ hasWorkflow: registry.hasWorkflow, console: this.options.console }),
        transport,
        { engineVersion: this.options.engineVersion ?? MinEngineVersion.DURABLE_EVICTION }
      );
    } catch (err) {
      transport.dispose();
      this.finish({
        error: `could not build the task context: ${errorMessage(err)}`,
        retry: false,
      });
      return;
    }

    this.transport = transport;
    this.ctx = ctx;
    this.state = 'running';

    const task = parentRunContextManager.runWithContext(
      {
        parentId: action.workflowRunId,
        parentTaskRunExternalId: action.taskRunExternalId,
        childIndex: 0,
        desiredWorkerId: '',
        signal: ctx.abortController.signal,
        durableContext: ctx,
      },
      () => Promise.resolve().then(() => runner(ctx))
    );

    try {
      const output = await task;
      const state = this.stateNow();

      if (state === 'running') {
        this.finish({ output: JSON.stringify(output === undefined ? {} : output) });
      } else if (state === 'engine-error') {
        this.finish({ error: errorMessage(this.engineError), retry: false });
      }
    } catch (err) {
      switch (this.stateNow()) {
        case 'running':
          this.out.error(
            `[hatchet] durable task ${actionId} (run ${action.workflowRunId}) failed: ${errorMessage(err)}`,
            err
          );
          this.finish({ error: errorMessage(err), retry: !(err instanceof NonRetryableError) });
          break;
        case 'engine-error':
          this.finish({ error: errorMessage(this.engineError ?? err), retry: false });
          break;
        default:
          // Evicted, superseded or closed: the task unwound because we aborted it.
          break;
      }
    } finally {
      transport.dispose();
    }
  }

  /** Evicts the invocation: evict_invocation, its ack, then done evicted as the last frame. */
  private async evict(reason: string): Promise<void> {
    if (this.state !== 'running' || !this.transport || !this.ctx) {
      return;
    }

    this.state = 'evicting';
    this.out.info(`[hatchet] durable task ${this.ctx.taskRunExternalId()}: ${reason}, evicting`);

    try {
      await this.transport.sendEvictInvocation(
        this.ctx.taskRunExternalId(),
        this.ctx.invocationCount,
        reason
      );
    } catch (err) {
      if (this.stateNow() === 'evicting') {
        this.out.warn(
          `[hatchet] eviction of ${this.ctx.taskRunExternalId()} failed: ${errorMessage(err)}`
        );
      }

      return;
    }

    if (this.stateNow() !== 'evicting') {
      return;
    }

    this.sendDone({ status: DONE_STATUS_EVICTED });
    this.state = 'evicted';
    this.abortTask(new TaskRunTerminatedError('evicted', reason));
    this.closeSocket();
  }

  private onServerEvict(reason: string): void {
    if (this.isTerminal()) {
      return;
    }

    this.out.info(
      `[hatchet] durable task ${this.ctx?.taskRunExternalId()}: evicted by the engine: ${reason}`
    );
    this.state = 'evicted';
    this.transport?.seal();
    this.abortTask(new TaskRunTerminatedError('evicted', reason));
  }

  private onEngineError(error: Error): void {
    if (this.isTerminal()) {
      return;
    }

    this.engineError = error;
    this.state = 'engine-error';
  }

  private onClose(code: number, reason: string): void {
    this.clearFirstFrameTimer();

    if (this.state === 'awaiting-first') {
      this.state = 'closed';
      this.firstFrame = undefined;
      // Resolve the pending first-frame wait with nothing to run.
      this.finishSilently();
      return;
    }

    if (this.isTerminal()) {
      return;
    }

    this.state = 'closed';
    this.transport?.seal();
    this.transport?.fail(
      new Error(`the operator closed the socket (code ${code}${reason ? `, ${reason}` : ''})`)
    );
    this.abortTask(new TaskRunTerminatedError(code === 4003 ? 'cancelled' : 'evicted', reason));
  }

  /** The state as of now; reading it through a call sidesteps narrowing across awaits. */
  private stateNow(): State {
    return this.state;
  }

  private isTerminal(): boolean {
    return (
      this.state === 'evicted' ||
      this.state === 'closed' ||
      this.state === 'finished' ||
      this.state === 'evicting'
    );
  }

  private abortTask(error: TaskRunTerminatedError): void {
    this.ctx?.abortController.abort(error);
    this.transport?.cleanupTaskState(
      this.ctx?.taskRunExternalId() ?? '',
      this.ctx?.invocationCount ?? 0
    );
  }

  private sendDone(done: {
    output?: string;
    error?: string;
    retry?: boolean;
    status?: string;
  }): void {
    try {
      this.options.socket.send(
        JSON.stringify({
          done: {
            ...(done.output !== undefined ? { output: done.output } : {}),
            ...(done.error !== undefined ? { error: done.error } : {}),
            ...(done.status ? { status: done.status } : {}),
            retry: done.retry ?? false,
          },
        })
      );
    } catch (err) {
      this.out.warn(`[hatchet] could not send the done frame: ${errorMessage(err)}`);
    }

    this.transport?.seal();
  }

  private finish(done: { output?: string; error?: string; retry?: boolean }): void {
    if (this.isTerminal()) {
      return;
    }

    this.sendDone(done);
    this.state = 'finished';
    this.closeSocket();
  }

  private finishSilently(): void {
    this.state = 'closed';
  }

  private closeSocket(): void {
    this.closeWith(1000, 'done');
  }

  private closeWith(code: number, reason: string): void {
    try {
      this.options.socket.close(code, reason);
    } catch {
      // Already closed by the operator.
    }
  }
}
