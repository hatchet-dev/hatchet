// pi-durable sessions as Hatchet tasks. One run of the task owns one pi-durable session: it opens the session's
// harness, applies what clients put in the session's inbox stream, and publishes the session's view to its view stream.
//
// The two layers split the work:
// - pi-durable checkpoints every model request and tool call in its own storage, and resumes them after a crash.
//   Nothing here is a Hatchet durable step, so nothing replays twice.
// - Hatchet decides where and when the session runs: one owner at a time (concurrency on the session id), a new
//   owner after a worker dies (retries), a run at the time of the next sleep (a scheduled run), and the sandboxes the
//   tools run in.
import { AsyncLocalStorage } from 'node:async_hooks';
import { setTimeout as sleep } from 'node:timers/promises';
import type { WatchHandle } from '@earendil-works/chord';
import { BACKGROUND_CONTEXT as context, type Context } from '@earendil-works/chord/context';
import {
  type AgentChange,
  type ConversationId,
  type ConversationView,
  defineDoc,
  type EntryId,
  type Harness,
  type HarnessInspection,
  type Report,
  ROOT_CONVERSATION_ID,
} from '@earendil-works/pi-durable';
import type { Sandbox } from '@hatchet-dev/sandbox';
import {
  ConcurrencyLimitStrategy,
  type Context as TaskContext,
  type Duration,
  type HatchetClient,
} from '@hatchet-dev/typescript-sdk/v1';
import { SandboxDoc } from './extension.js';
import { InboxMessage, type RejectedMessage, type SessionView, sessionTopics } from './protocol.js';
import { DEFAULT_WORKSPACE, latestSnapshot } from './sandbox-env.js';
import { summarize } from './view.js';

export type SessionTaskInput = { sessionId: string };
export type SessionTaskOutput = {
  sessionId: string;
  recoveredTasks: number;
  nextWake: number | null;
};

export type SessionTaskOptions = {
  /** The task's name, which also prefixes the session's stream topics. Default `pi-session`. */
  name?: string;
  /**
   * Opens the session's harness. Give it `env: sandboxEnvs(hatchet, …)` so tools run in the conversation's sandbox,
   * and a registry with `sandboxCoding()` or your own tools.
   *
   * Its storage decides how far failover reaches: a new owner opens the same storage, so storage on a worker's disk
   * only fails over to a worker on the same machine.
   */
  openHarness: (
    sessionId: string,
    onReport: (report: Report) => void,
    context: Context
  ) => Promise<Harness>;
  /** The root conversation's agent: model, thinking level, working directory. */
  agent: AgentChange;
  sandbox: {
    /** The image every conversation's sandbox runs. */
    template: string;
    /** The sandbox's workspace directory. Default `/workspace`. */
    workspace?: string;
    /** A sandbox with no calls for this long stops; the next call restarts it from its snapshot. Default 30 minutes. */
    idleTimeoutMs?: number;
    /** Fills a conversation's first sandbox, for example with a repository. Forks start from a snapshot instead. */
    seed?: (sandbox: Sandbox, sessionId: string) => Promise<void>;
  };
  /** Every conversation of a session is aborted once the session has spent this much. Default: no cap. */
  maxSessionCostUsd?: number;
  /** A host with nothing left to do waits this long for the next message before it hands the session back. Default 20 s. */
  idleGraceMs?: number;
  /** A ready task that has not started after this long means the owner is stuck; a new owner takes over. Default 15 s. */
  stallMs?: number;
  /** New owners after a worker dies. Default 3. */
  retries?: number;
  /** How long one owner may hold the session. Default 60 minutes. */
  executionTimeout?: Duration;
  /** Logs the harness's task states to the run whenever they change. */
  debug?: boolean;
};

const PUBLISH_DELAY_MS = 200;

const checkpointEvery32 = (_value: unknown, _ops: unknown, info: { deltasSinceBase: number }) =>
  info.deltasSinceBase >= 32;

/**
 * How long a host that found nothing to do keeps reading its inbox. A client publishes a message and then wakes the
 * session; the run can start before the inbox reader delivers that message or the topic's metadata shows it.
 */
const WAKE_GRACE_MS = 2_000;

/** How many rejected messages the session keeps for its view. */
const MAX_REJECTED = 20;

/** What the host has applied. The inbox cursor commits with the work, so a new owner resumes after it. */
export const HostDoc = defineDoc<{
  conversations: number[];
  forks: Record<string, number>;
  inboxCursor?: string;
  rejected?: RejectedMessage[];
}>({
  kind: 'hatchet.host',
  version: 1,
  scope: 'session',
  initial: () => ({ conversations: [], forks: {} }),
  checkpointWhen: checkpointEvery32,
});

type LogLevel = 'info' | 'debug' | 'warn' | 'error';

const decoder = new TextDecoder();

/**
 * Runs a function in the async context of this module's import, where no task is running. Inside a task, the SDK turns
 * every run it starts into a child of that task, numbered in start order. A retry keeps the task run's id and numbers
 * from zero again, so the engine would answer each new sandbox call with the previous attempt's child of that number.
 * Sandbox calls, wakes and scheduled runs are not children of the session: they run outside the task's context.
 */
const outsideTask = AsyncLocalStorage.snapshot();

/** Defines the task that hosts sessions. Register it on a worker; start or wake a session with `SessionClient`. */
export function defineSessionTask(hatchet: HatchetClient, options: SessionTaskOptions) {
  const name = options.name ?? 'pi-session';
  return hatchet.task<SessionTaskInput, SessionTaskOutput>({
    name,
    retries: options.retries ?? 3,
    executionTimeout: options.executionTimeout ?? '60m',
    concurrency: {
      expression: 'input.sessionId',
      maxRuns: 1,
      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    },
    fn: (input, ctx) =>
      outsideTask(() => new SessionHost(hatchet, options, name, input.sessionId, ctx).run()),
  });
}

class SessionHost {
  private harness!: Harness;
  private readonly views = new Map<number, WatchHandle<ConversationView>>();
  private readonly startedAt = Date.now();
  private readonly topics: ReturnType<typeof sessionTopics>;
  private state: SessionView['host']['state'] = 'starting';
  private recoveredTasks = 0;
  private appliedCursor: string | undefined;
  /** When the inbox reader last applied a message. A run it submitted may not be visible to `inspect()` yet. */
  private lastApplied = 0;
  private publishTimer: NodeJS.Timeout | undefined;
  private publishing = Promise.resolve();
  private closing = false;
  private costCapHit = false;
  private readonly readySince = new Map<number, number>();

  constructor(
    private readonly hatchet: HatchetClient,
    private readonly options: SessionTaskOptions,
    private readonly taskName: string,
    private readonly sessionId: string,
    private readonly ctx: TaskContext<SessionTaskInput>
  ) {
    this.topics = sessionTopics(taskName, sessionId);
  }

  /** Logs to the task run, so the line shows up on the run in the dashboard. Hatchet keeps 1000 lines per task. */
  private log(message: string, level: LogLevel = 'info') {
    Promise.resolve(this.ctx.logger[level](message)).catch((error) =>
      console.error(
        `[${this.taskName} ${this.sessionId}] could not send a log line: ${String(error)}`
      )
    );
  }

  async run(): Promise<SessionTaskOutput> {
    const { signal } = this.ctx.abortController;
    this.harness = await this.options.openHarness(
      this.sessionId,
      (report) => this.log(`${report.kind}: ${String(report.error)}`, 'error'),
      context
    );

    // A crash leaves the last owner's unfinished tasks live in storage; `resume()` continues them.
    this.recoveredTasks = (await this.harness.inspect(context)).tasks.length;
    const isNew = (await this.harness.conversation(ROOT_CONVERSATION_ID, context)) === undefined;
    const root = await this.harness.root(context, { agent: this.options.agent });
    await root.commit(async (tx) => {
      const host = await tx.doc(HostDoc);
      if (!host.conversations.includes(root.id)) host.conversations.push(root.id);
    }, context);
    const host = await this.harness.snapshot(HostDoc, context);
    this.appliedCursor = host?.inboxCursor;
    this.log(
      `attempt ${this.ctx.retryCount()}, ${this.recoveredTasks} tasks to recover, new: ${isNew}`
    );

    for (const id of host?.conversations ?? [root.id]) await this.watch(id as ConversationId);
    await this.ensureSandbox(root.id);
    this.state = 'running';
    this.schedulePublish();
    this.harness.resume();

    const inbox = new AbortController();
    signal.addEventListener('abort', () => inbox.abort(), { once: true });
    const reading = this.readInbox(inbox.signal);
    const { stalled } = await this.untilIdle(signal, isNew || this.recoveredTasks > 0);
    inbox.abort();
    await reading;

    const wake = this.harness.wake();
    if (!signal.aborted) {
      // A sleeping task, such as a retry backoff, needs a process at its deadline: a scheduled run is that process.
      if (wake.at !== undefined) {
        await this.hatchet.scheduled.create(this.taskName, {
          triggerAt: new Date(wake.at),
          input: { sessionId: this.sessionId },
        });
        this.log(`scheduled a wakeup at ${new Date(wake.at).toISOString()}`);
      }
      // Queues behind this run on the session's concurrency key, and recovers the stuck task when it opens the session.
      if (stalled) await this.hatchet.runNoWait(this.taskName, { sessionId: this.sessionId }, {});
    }

    this.state = 'stopped';
    this.closing = true;
    clearTimeout(this.publishTimer);
    await this.publish({ final: true });
    await Promise.all([...this.views.values()].map((view) => view.stop()));
    await this.harness.close(context);
    return {
      sessionId: this.sessionId,
      recoveredTasks: this.recoveredTasks,
      nextWake: wake.at ?? null,
    };
  }

  /** Returns once nothing has needed this process for the idle grace, or once a ready task stalls. */
  private async untilIdle(signal: AbortSignal, didWork: boolean): Promise<{ stalled: boolean }> {
    const idleGraceMs = this.options.idleGraceMs ?? 20_000;
    let lastBusy = Date.now();
    let lastState: string | undefined;
    while (!signal.aborted) {
      const inspection = await this.harness.inspect(context);
      if (this.options.debug) {
        const state = `wake ${JSON.stringify(this.harness.wake())} scheduling ${inspection.scheduling} tasks ${describe(inspection)}`;
        if (state !== lastState) this.log(state, 'debug');
        lastState = state;
      }
      if (this.lastApplied > lastBusy) {
        didWork = true;
        lastBusy = this.lastApplied;
      }
      if (this.stalled(inspection)) {
        this.log(
          `a ready task did not start in time, handing the session over: ${describe(inspection)}`,
          'warn'
        );
        return { stalled: true };
      }
      const busy =
        this.harness.wake().active ||
        inspection.tasks.some((t) => t.state.kind === 'pending' || t.state.kind === 'running') ||
        !(await this.inboxDrained());
      if (busy) {
        didWork = true;
        lastBusy = Date.now();
        await this.enforceCostCap();
      } else if (Date.now() - lastBusy >= (didWork ? idleGraceMs : WAKE_GRACE_MS)) {
        break;
      }
      await sleep(1000);
    }
    return { stalled: false };
  }

  /** A new owner recovers a ready task that never started when it opens the session, so the host hands over. */
  private stalled(inspection: HarnessInspection): boolean {
    const stallMs = this.options.stallMs ?? 15_000;
    const ready = new Set(
      inspection.tasks.filter((t) => t.state.kind === 'pending').map((t) => t.record.id as number)
    );
    for (const id of this.readySince.keys()) if (!ready.has(id)) this.readySince.delete(id);
    for (const id of ready) if (!this.readySince.has(id)) this.readySince.set(id, Date.now());
    return [...this.readySince.values()].some((since) => Date.now() - since > stallMs);
  }

  private async watch(id: ConversationId) {
    if (this.views.has(id)) return;
    const conversation = await this.harness.conversation(id, context);
    if (!conversation) return;
    // The view holds a window over the newest entries (50 to 100), not the whole transcript.
    const watch = await conversation.watch(context);
    this.views.set(id, watch);
    watch.start(async () => this.schedulePublish());
  }

  private createSandbox(from: string | undefined, metadata: Record<string, string>) {
    return this.hatchet.sandbox.create({
      template: this.options.sandbox.template,
      from,
      idleTimeoutMs: this.options.sandbox.idleTimeoutMs ?? 30 * 60_000,
      metadata: { piSession: this.sessionId, ...metadata },
    });
  }

  /** A conversation's first input needs a sandbox it owns: a fresh, seeded one, or one from its snapshot. */
  private async ensureSandbox(id: ConversationId) {
    const current = await this.harness.snapshot(SandboxDoc, id, context);
    if (current?.sandboxId && current.owner === id) return;
    const sandbox = await this.createSandbox(current?.snapshotId, { conversation: String(id) });
    if (!current?.snapshotId) await this.options.sandbox.seed?.(sandbox, this.sessionId);
    const snapshotId = await latestSnapshot(this.hatchet, sandbox.sandboxId);
    const conversation = (await this.harness.conversation(id, context))!;
    await conversation.commit(async (tx) => {
      Object.assign(await tx.doc(SandboxDoc, id), {
        owner: id,
        sandboxId: sandbox.sandboxId,
        snapshotId,
      });
    }, context);
    this.log(
      `conversation ${id} runs in sandbox ${sandbox.sandboxId} (workspace ${this.options.sandbox.workspace ?? DEFAULT_WORKSPACE})`
    );
  }

  private async readInbox(signal: AbortSignal) {
    while (!signal.aborted) {
      try {
        for await (const event of this.hatchet.streams.events(this.topics.inbox, {
          cursor: this.appliedCursor,
          signal,
        })) {
          const parsed = InboxMessage.safeParse(JSON.parse(decoder.decode(event.payload)));
          let rejected: RejectedMessage | undefined;
          if (!parsed.success)
            this.log(`skipped an invalid inbox message: ${parsed.error.message}`, 'warn');
          else {
            const message = parsed.data;
            const reason = await this.apply(message).catch(
              (error) => `could not apply it: ${String(error)}`
            );
            if (reason) {
              this.log(
                `rejected ${message.type} ${message.requestId} for conversation ${message.conversationId}: ${reason}`,
                'warn'
              );
              const { requestId, type, conversationId } = message;
              rejected = { requestId, type, conversationId, reason, at: Date.now() };
            }
          }

          const cursor = String(event.cursor);
          const root = (await this.harness.conversation(ROOT_CONVERSATION_ID, context))!;
          await root.commit(async (tx) => {
            const host = await tx.doc(HostDoc);
            host.inboxCursor = cursor;
            if (rejected) host.rejected = [...(host.rejected ?? []), rejected].slice(-MAX_REJECTED);
          }, context);
          if (rejected) this.schedulePublish();
          this.appliedCursor = cursor;
          this.lastApplied = Date.now();
        }
      } catch (error) {
        if (!signal.aborted) this.log(`inbox stream ended: ${String(error)}`, 'warn');
      }
      // the engine may hang up a subscription; read again from the committed cursor
      if (!signal.aborted) await sleep(500);
    }
  }

  private async inboxDrained(): Promise<boolean> {
    try {
      const latest = (await this.hatchet.streams.topicMetadata(this.topics.inbox)).latestCursor;
      return latest === undefined || String(latest) === this.appliedCursor;
    } catch {
      return true; // nothing was ever published to the inbox
    }
  }

  /**
   * Applies one inbox message. Returns why it could not, for the view's `rejected` list. Redelivered messages are
   * harmless: inputs carry their request id, and forks are looked up by theirs.
   */
  private async apply(message: InboxMessage): Promise<string | undefined> {
    const conversation = await this.harness.conversation(
      message.conversationId as ConversationId,
      context
    );
    if (!conversation) return `conversation ${message.conversationId} does not exist`;
    switch (message.type) {
      case 'input':
        await this.ensureSandbox(conversation.id);
        await conversation.submit(
          {
            type: 'input',
            content: message.text,
            requestId: message.requestId,
            whenBusy: message.whenBusy,
          },
          context
        );
        return;
      case 'abort':
        void conversation.abort(context);
        return;
      case 'fork': {
        const host = await this.harness.snapshot(HostDoc, context);
        if (host?.forks[message.requestId] !== undefined) return;
        const at = message.entryId as EntryId;
        const asOf = await this.harness.snapshotAsOf(SandboxDoc, conversation.id, at, context);
        const sandbox = await this.createSandbox(asOf?.snapshotId, {
          forkOf: String(conversation.id),
        });
        const fork = await conversation.fork(
          at,
          {
            ownership: { kind: 'ownerless' },
            init: async (tx, id) => {
              Object.assign(await tx.doc(SandboxDoc, id), {
                owner: id,
                sandboxId: sandbox.sandboxId,
              });
              const hostDoc = await tx.doc(HostDoc);
              hostDoc.forks[message.requestId] = id;
              hostDoc.conversations.push(id);
            },
          },
          context
        );
        this.log(
          `forked conversation ${conversation.id} at entry ${at} into ${fork.id}, sandbox ${sandbox.sandboxId}`
        );
        await this.watch(fork.id);
        return;
      }
    }
  }

  private async enforceCostCap() {
    const cap = this.options.maxSessionCostUsd;
    if (cap === undefined || this.costCapHit) return;
    const usage = await this.harness.usage(context);
    const cost = Object.values(usage.models).reduce((sum, u) => sum + u.cost.total, 0);
    if (cost < cap) return;
    this.costCapHit = true;
    this.log(
      `session spent $${cost.toFixed(2)}, over the $${cap} cap: aborting every conversation`,
      'warn'
    );
    for (const id of this.views.keys()) {
      const conversation = await this.harness.conversation(id as ConversationId, context);
      if (conversation) void conversation.abort(context);
    }
  }

  private schedulePublish() {
    if (this.publishTimer || this.closing) return;
    this.publishTimer = setTimeout(() => {
      this.publishTimer = undefined;
      void this.publish();
    }, PUBLISH_DELAY_MS);
  }

  /** Publishes run one at a time, so a reset never lands between another publish's reset and its message. */
  private publish({ final = false } = {}): Promise<void> {
    this.publishing = this.publishing.then(async () => {
      if (this.closing && !final) return;
      try {
        const view = await this.buildView();
        await this.hatchet.streams.reset(this.topics.view);
        await this.hatchet.streams.publish(this.topics.view, JSON.stringify(view));
      } catch (error) {
        this.log(`could not publish the view: ${String(error)}`, 'error');
      }
    });
    return this.publishing;
  }

  private async buildView(): Promise<SessionView> {
    const conversations = [];
    for (const [id, state] of this.views) {
      const sandbox =
        (await this.harness.snapshot(SandboxDoc, id as ConversationId, context)) ?? {};
      conversations.push(summarize(state.value, sandbox));
    }
    return {
      sessionId: this.sessionId,
      host: {
        runId: this.ctx.workflowRunId(),
        attempt: this.ctx.retryCount(),
        state: this.state,
        startedAt: this.startedAt,
      },
      recoveredTasks: this.recoveredTasks,
      conversations,
      rejected: (await this.harness.snapshot(HostDoc, context))?.rejected ?? [],
      publishedAt: Date.now(),
    };
  }
}

const describe = (inspection: HarnessInspection) =>
  JSON.stringify(inspection.tasks.map((t) => [t.record.id, t.record.kind, t.state.kind]));
