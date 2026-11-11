import { HatchetClient, JsonObject } from '@hatchet-dev/typescript-sdk/v1';
import { z } from 'zod';
import { Files } from './files.js';
import { setupTelemetry } from './otel.js';
import {
  commandOutput,
  metadata,
  sandboxInfo,
  turnInput,
  turnResult,
  type TurnInput,
  type TurnResult,
} from './wire.js';

const TASK_NAMES = ['hatchet-sandbox', 'brickie-sandbox'];
const taskNames = new WeakMap<HatchetClient, Promise<string>>();

async function registered(client: HatchetClient, name: string): Promise<boolean> {
  const namespace = client.config.namespace ?? '';
  const full = name.startsWith(namespace) ? name : `${namespace}${name}`;
  for (let offset = 0; ;) {
    const { rows = [] } = await client.workflows.list({ name: full, limit: 100, offset });
    if (rows.some((w) => w.name === full)) return true;
    if (rows.length === 0) return false;
    offset += rows.length;
  }
}

async function probe(client: HatchetClient): Promise<string | undefined> {
  for (const name of TASK_NAMES) {
    if (await registered(client, name)) return name;
  }
  return undefined;
}

function resolveTaskName(client: HatchetClient): Promise<string> {
  let pending = taskNames.get(client);
  if (!pending) {
    pending = probe(client).then(
      (name) => {
        if (!name) taskNames.delete(client);
        return name ?? TASK_NAMES[TASK_NAMES.length - 1];
      },
      (err) => {
        taskNames.delete(client);
        throw err;
      }
    );
    taskNames.set(client, pending);
  }
  return pending;
}

const positiveInt = z.int().positive();
const nonEmpty = z.string().min(1);

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

const commandResult = commandOutput.extend(
  turnResult.pick({ snapshotId: true, timings: true }).shape
);

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
  /** Machine size. Providers that cannot honour a field fail the create with their own message. Providers that do not know the option ignore it. */
  resources?: Resources;
}

export interface Resources {
  /** Whole cores. */
  vcpu?: number;
  /** A number is MiB. A string takes a Kubernetes quantity with a unit, such as `'512Mi'`, `'4Gi'` or `'8G'`. A string without a unit is rejected. */
  memory?: number | string;
  /** Same format as `memory`. */
  disk?: number | string;
}

const UNITS: Record<string, number> = {
  K: 1e3,
  M: 1e6,
  G: 1e9,
  T: 1e12,
  Ki: 2 ** 10,
  Mi: 2 ** 20,
  Gi: 2 ** 30,
  Ti: 2 ** 40,
};
const QUANTITY = /^(\d+(?:\.\d+)?)\s*([KMGT]i?)B?$/;

const mib = z.union(
  [
    positiveInt,
    z
      .string()
      .trim()
      .regex(QUANTITY)
      .transform((s) => {
        const [, n, unit] = QUANTITY.exec(s)!;
        return Math.ceil((Number(n) * UNITS[unit]) / UNITS.Mi);
      })
      .pipe(z.int().min(1)),
  ],
  {
    error:
      "expected a positive whole number of MiB or a quantity with a unit like '512Mi', '4Gi' or '8G'",
  }
);

const resources = z
  .strictObject({
    vcpu: positiveInt.optional(),
    memory: mib.optional(),
    disk: mib.optional(),
  })
  .refine(
    (r) => r.vcpu !== undefined || r.memory !== undefined || r.disk !== undefined,
    'set at least one of vcpu, memory, disk'
  )
  .transform(({ vcpu, memory, disk }) => ({ vcpu, memoryMib: memory, diskMib: disk }));

const sandboxOpts = z
  .strictObject({
    id: z
      .string()
      .regex(/^[a-z0-9-]+$/, 'lowercase letters, digits, and hyphens only')
      .optional(),
    template: nonEmpty.optional(),
    from: nonEmpty.optional(),
    metadata: metadata.optional(),
    idleTimeoutMs: positiveInt.optional(),
    inheritParentMetadata: z.boolean().optional(),
    provider: nonEmpty.optional(),
    resources: resources.optional(),
  })
  .refine((o) => !o.inheritParentMetadata || o.from !== undefined, {
    error: 'inheritParentMetadata needs from',
    path: ['inheritParentMetadata'],
  });

export interface ExecOpts {
  /** Kill the command after this long. The result then has `exitCode` 124. */
  timeoutMs?: number;
}

const execOpts = z.strictObject({ timeoutMs: positiveInt.optional() });

export interface ConnectOpts {
  /** Image to use if the sandbox has to be started again. Defaults to the provider's image. */
  template?: string;
}

const connectOpts = z.strictObject({ template: nonEmpty.optional() });

export interface SandboxListOpts {
  /** Only the sandbox with this ID. */
  sandboxId?: string;
  /** Only sandboxes whose metadata contains every one of these pairs. */
  metadata?: Record<string, string>;
}

const listOpts = z.strictObject({
  sandboxId: nonEmpty.optional(),
  metadata: metadata.optional(),
});

const deleteOpts = listOpts.refine(
  (o) => o.sandboxId !== undefined || Object.keys(o.metadata ?? {}).length > 0,
  'delete needs a sandboxId or a metadata filter'
);

export interface SandboxInfo {
  sandboxId: string;
  metadata?: Record<string, string>;
  /** True while the sandbox is running. A sandbox that is not live is started again by `connect`. */
  live: boolean;
  latestSnapshot?: string;
  createdAt: Date;
  lastUsed?: Date;
}

type RunTurn = (input: TurnInput, opts?: { fresh?: boolean }) => Promise<TurnResult>;

function turnRunner(client: HatchetClient): RunTurn {
  const resolved = resolveTaskName(client);
  return async (raw, opts) => {
    const input = turnInput.parse(raw);
    const payload = input as JsonObject;
    const name = await resolved;
    const additionalMetadata = {
      'hatchet.sandbox.fn': input.fn,
      ...(input.sandboxId ? { 'hatchet.sandbox.id': input.sandboxId } : {}),
      ...(input.image ? { 'hatchet.sandbox.image': input.image } : {}),
      ...(input.snapshotId ? { 'hatchet.sandbox.from': input.snapshotId } : {}),
    };
    const result = opts?.fresh
      ? await (
          await client.admin.runWorkflow<JsonObject, Record<string, JsonObject>>(name, payload, {
            additionalMetadata,
          })
        ).result()
      : await client
          .workflow<JsonObject, Record<string, JsonObject>>({ name })
          .run(payload, { additionalMetadata });
    return turnResult.parse(result[name] ?? result);
  };
}

/** One sandbox. Every method is one Hatchet task on the tenant's sandbox provider. */
export class Sandbox {
  /** File operations on the workspace. */
  readonly files: Files;

  private constructor(
    private readonly run: RunTurn,
    readonly sandboxId: string,
    private readonly template?: string
  ) {
    this.files = new Files((fn, input, fresh) => this.turn(fn, input, fresh));
  }

  /** Starts a new sandbox. Prefer `hatchet.sandbox.create`. */
  static async create(client: HatchetClient, opts: SandboxOpts = {}): Promise<Sandbox> {
    const o = sandboxOpts.parse(opts);
    await setupTelemetry();
    const run = turnRunner(client);
    const created = await run({
      fn: '$create',
      input: {},
      sandboxId: o.id,
      image: o.template,
      snapshotId: o.from,
      metadata: o.provider ? { ...o.metadata, provider: o.provider } : o.metadata,
      idleTtlMs: o.idleTimeoutMs,
      inheritMetadata: o.inheritParentMetadata,
      resources: o.resources,
    });
    return new Sandbox(run, created.sandboxId, o.template);
  }

  /** Attaches to a sandbox by ID. Prefer `hatchet.sandbox.connect`. */
  static async connect(
    client: HatchetClient,
    sandboxId: string,
    opts: ConnectOpts = {}
  ): Promise<Sandbox> {
    const id = nonEmpty.parse(sandboxId);
    const o = connectOpts.parse(opts);
    await setupTelemetry();
    const run = turnRunner(client);
    await run({ fn: '$connect', input: {}, sandboxId: id, image: o.template });
    return new Sandbox(run, id, o.template);
  }

  /** Lists sandboxes on the provider. Prefer `hatchet.sandbox.list`. */
  static async list(client: HatchetClient, opts: SandboxListOpts = {}): Promise<SandboxInfo[]> {
    const o = listOpts.parse(opts);
    const result = await turnRunner(client)({ fn: '$list', input: o });
    return sandboxInfo.array().parse(result.output.sandboxes ?? []);
  }

  /** Deletes every sandbox matching the filter, with all their snapshots and files. Prefer `hatchet.sandbox.delete`. */
  static async delete(client: HatchetClient, opts: SandboxListOpts): Promise<string[]> {
    deleteOpts.parse(opts);
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
    const input = { cmd: nonEmpty.parse(cmd), ...execOpts.parse(opts) };
    return commandResult.parse(await this.turn('$command', input));
  }

  /** Removes the sandbox. Its snapshots stay; `connect` can bring it back. */
  async kill(): Promise<void> {
    await this.turn('$destroy', {});
  }

  /** Removes the sandbox and deletes every snapshot and file the provider stored for it. `connect` fails afterwards. */
  async delete(): Promise<void> {
    await this.turn('$delete', {});
  }

  private async turn(fn: string, input: JsonObject, fresh = false): Promise<JsonObject> {
    const result = await this.run(
      { sandboxId: this.sandboxId, image: this.template, fn, input },
      { fresh }
    );
    return {
      ...result.output,
      snapshotId: result.snapshotId,
      timings: result.timings,
    } as JsonObject;
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
