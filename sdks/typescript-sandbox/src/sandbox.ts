import { HatchetClient, JsonObject } from '@hatchet-dev/typescript-sdk/v1';
import { Files } from './files.js';
import { setupTelemetry } from './otel.js';

const SANDBOX_TASK_NAME = 'brickie-sandbox';

interface TurnInput extends JsonObject {
  sandboxId?: string;
  image?: string;
  fn: string;
  input: JsonObject;
  snapshotId?: string;
  metadata?: Record<string, string>;
  idleTtlMs?: number;
  inheritMetadata?: boolean;
}

interface TurnResult extends JsonObject {
  output: JsonObject;
  sandboxId: string;
  snapshotId?: string;
  timings?: Record<string, number>;
}

export interface Timings {
  /** Snapshot made after this turn. Absent for turns that do not change the workspace. */
  snapshotId?: string;
  /** Milliseconds per phase: `claim`, `restore`, `exec`, `checkpoint`. */
  timings?: Record<string, number>;
}

export interface CommandResult extends Timings {
  stdout: string;
  stderr: string;
  /** A non-zero code is a normal result, not an error. */
  exitCode: number;
}

export interface SandboxOpts {
  /** Sandbox ID to use instead of a generated one. Lowercase letters, digits, and hyphens; must be new on the provider. */
  id?: string;
  /** Container image. Defaults to the provider's image. */
  template?: string;
  /** Snapshot to start from. Two sandboxes with the same `from` are a fork. */
  from?: string;
  /** Key-value tags, used to find the sandbox again with {@link SandboxNamespace.list}. */
  metadata?: Record<string, string>;
  /** Milliseconds without a turn before the provider removes the sandbox. Capped by the provider. */
  idleTimeoutMs?: number;
  /** With `from`, start from the metadata of the sandbox that made that snapshot. Pairs given in `metadata` take precedence over inherited ones. */
  inheritParentMetadata?: boolean;
  /** Name of the sandbox provider to create on. Required when the tenant has more than one; with `from`, it must be the provider that holds the snapshot. */
  provider?: string;
}

export interface ExecOpts {
  /** Kill the command after this long. The result then has `exitCode` 124. */
  timeoutMs?: number;
}

export interface ConnectOpts {
  /** Image to use if the sandbox has to be started again. Defaults to the provider's image. */
  template?: string;
}

export interface SandboxListOpts {
  /** Only the sandbox with this ID. */
  sandboxId?: string;
  /** Only sandboxes whose metadata contains every one of these pairs. */
  metadata?: Record<string, string>;
}

export interface SandboxInfo {
  sandboxId: string;
  metadata?: Record<string, string>;
  /** True while the sandbox is running. A sandbox that is not live is started again by `connect`. */
  live: boolean;
  latestSnapshot?: string;
  createdAt: Date;
  lastUsed?: Date;
}

type RunTurn = (input: TurnInput) => Promise<TurnResult>;

function turnRunner(client: HatchetClient): RunTurn {
  const task = client.workflow<TurnInput, Record<string, TurnResult>>({ name: SANDBOX_TASK_NAME });
  return async (input) => {
    const result = await task.run(input, {
      additionalMetadata: {
        'brickie.fn': input.fn,
        ...(input.sandboxId ? { 'brickie.sandbox': input.sandboxId } : {}),
        ...(input.image ? { 'brickie.image': input.image } : {}),
        ...(input.snapshotId ? { 'brickie.from': input.snapshotId } : {}),
      },
    });
    return result[SANDBOX_TASK_NAME] ?? (result as unknown as TurnResult);
  };
}

/** One sandbox. Every method is one `brickie-sandbox` Hatchet task. */
export class Sandbox {
  /** File operations on the workspace. */
  readonly files: Files;

  private constructor(
    private readonly run: RunTurn,
    readonly sandboxId: string,
    private readonly template?: string
  ) {
    this.files = new Files((fn, input) => this.turn(fn, input));
  }

  /** Starts a new sandbox. Prefer `hatchet.sandbox.create`. */
  static async create(client: HatchetClient, opts: SandboxOpts = {}): Promise<Sandbox> {
    void setupTelemetry();
    const run = turnRunner(client);
    const metadata = opts.provider ? { ...opts.metadata, provider: opts.provider } : opts.metadata;
    const created = await run({
      fn: '$create',
      input: {},
      ...(opts.id ? { sandboxId: opts.id } : {}),
      ...(opts.template ? { image: opts.template } : {}),
      ...(opts.from ? { snapshotId: opts.from } : {}),
      ...(metadata ? { metadata } : {}),
      ...(opts.idleTimeoutMs ? { idleTtlMs: opts.idleTimeoutMs } : {}),
      ...(opts.inheritParentMetadata ? { inheritMetadata: true } : {}),
    });
    return new Sandbox(run, created.sandboxId, opts.template);
  }

  /** Attaches to a sandbox by ID. Prefer `hatchet.sandbox.connect`. */
  static async connect(
    client: HatchetClient,
    sandboxId: string,
    opts: ConnectOpts = {}
  ): Promise<Sandbox> {
    void setupTelemetry();
    const run = turnRunner(client);
    await run({
      fn: '$connect',
      input: {},
      sandboxId,
      ...(opts.template ? { image: opts.template } : {}),
    });
    return new Sandbox(run, sandboxId, opts.template);
  }

  /** Lists sandboxes on the provider. Prefer `hatchet.sandbox.list`. */
  static async list(client: HatchetClient, opts: SandboxListOpts = {}): Promise<SandboxInfo[]> {
    const result = await turnRunner(client)({
      fn: '$list',
      input: {
        ...(opts.sandboxId ? { sandboxId: opts.sandboxId } : {}),
        ...(opts.metadata ? { metadata: opts.metadata } : {}),
      },
    });
    const raw = (result.output.sandboxes ?? []) as Array<
      Omit<SandboxInfo, 'createdAt' | 'lastUsed'> & { createdAt: string; lastUsed?: string }
    >;
    return raw.map((s) => ({
      ...s,
      createdAt: new Date(s.createdAt),
      lastUsed: s.lastUsed ? new Date(s.lastUsed) : undefined,
    }));
  }

  /** Deletes every sandbox matching the filter, with all their snapshots and files. Prefer `hatchet.sandbox.delete`. */
  static async delete(client: HatchetClient, opts: SandboxListOpts): Promise<string[]> {
    if (!opts.sandboxId && Object.keys(opts.metadata ?? {}).length === 0)
      throw new Error('delete needs a sandboxId or a metadata filter');
    const run = turnRunner(client);
    const ids: string[] = [];
    for (const { sandboxId } of await Sandbox.list(client, opts)) {
      await run({ fn: '$delete', input: {}, sandboxId });
      ids.push(sandboxId);
    }
    return ids;
  }

  /** Runs a shell command with `sh -c` in the workspace and makes a snapshot. A timeout gives `exitCode` 124, not an error. */
  async exec(cmd: string, opts: ExecOpts = {}): Promise<CommandResult> {
    return this.turn<CommandResult>('$command', {
      cmd,
      ...(opts.timeoutMs ? { timeoutMs: opts.timeoutMs } : {}),
    });
  }

  /** Removes the sandbox. Its snapshots stay; `connect` can bring it back. */
  async kill(): Promise<void> {
    await this.turn('$destroy', {});
  }

  /** Removes the sandbox and deletes every snapshot and file the provider stored for it. `connect` fails afterwards. */
  async delete(): Promise<void> {
    await this.turn('$delete', {});
  }

  private async turn<T = JsonObject>(fn: string, input: JsonObject): Promise<T> {
    const result = await this.run({
      sandboxId: this.sandboxId,
      ...(this.template ? { image: this.template } : {}),
      fn,
      input,
    });
    return {
      ...result.output,
      snapshotId: result.snapshotId,
      timings: result.timings,
    } as unknown as T;
  }
}

/** `hatchet.sandbox`, added to `HatchetClient` by importing `@hatchet-dev/sandbox`. */
export interface SandboxNamespace {
  /** Starts a new sandbox. Inside a durable task this is a durable step. */
  create(opts?: SandboxOpts): Promise<Sandbox>;
  /** Attaches to a sandbox by ID. If it is not running, the provider starts it from its latest snapshot. */
  connect(sandboxId: string, opts?: ConnectOpts): Promise<Sandbox>;
  /** Lists sandboxes the provider knows, filtered by ID or metadata. */
  list(opts?: SandboxListOpts): Promise<SandboxInfo[]>;
  /** Deletes the sandboxes matching the filter, with every snapshot and file stored for them. Returns their IDs. */
  delete(opts: SandboxListOpts): Promise<string[]>;
}

declare module '@hatchet-dev/typescript-sdk/v1' {
  interface HatchetClient {
    readonly sandbox: SandboxNamespace;
  }
}

Object.defineProperty(HatchetClient.prototype, 'sandbox', {
  configurable: true,
  get(this: HatchetClient): SandboxNamespace {
    return {
      create: (opts) => Sandbox.create(this, opts),
      connect: (sandboxId, opts) => Sandbox.connect(this, sandboxId, opts),
      list: (opts) => Sandbox.list(this, opts),
      delete: (opts) => Sandbox.delete(this, opts),
    };
  },
});
