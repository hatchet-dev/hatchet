// A sandbox provider for the e2e tests: the `hatchet-sandbox` task, served from folders on this machine instead of
// containers. It speaks the same task protocol as brickie, so the sandbox SDK and SandboxEnv run unchanged.
import { spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import { turnInput, type TurnInput } from '@hatchet-dev/sandbox/wire';
import type { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';

/** The workspace path commands use. There is no container to mount the folder there, so commands are rewritten. */
const WORKSPACE = '/workspace';
const WORKSPACE_IN_COMMAND = /\/workspace(?=[/'"\s;&|)]|$)/g;

type SandboxRecord = {
  sandboxId: string;
  metadata: Record<string, string>;
  live: boolean;
  latestSnapshot?: string;
  createdAt: Date;
  lastUsed?: Date;
};

type Entry = {
  name: string;
  path: string;
  type: 'file' | 'dir';
  size: number;
  mode: number;
  permissions: string;
  modifiedTime: string;
};

/** What the tests can check about the provider after the fact. */
export type ProviderLog = {
  commands: { sandboxId: string; cmd: string; exitCode?: number; killed: boolean }[];
};

class FsError extends Error {
  constructor(
    readonly code: string,
    message: string
  ) {
    super(message);
  }
}

export function fakeSandboxProvider(hatchet: HatchetClient, root: string) {
  const sandboxes = new Map<string, SandboxRecord>();
  const log: ProviderLog = { commands: [] };
  const dirOf = (sandboxId: string) => path.join(root, 'sandboxes', sandboxId);
  const snapshotDir = (snapshotId: string) => path.join(root, 'snapshots', snapshotId);

  async function snapshot(record: SandboxRecord): Promise<string> {
    const snapshotId = `snap-${randomUUID()}`;
    await fs.cp(dirOf(record.sandboxId), snapshotDir(snapshotId), { recursive: true });
    record.latestSnapshot = snapshotId;
    return snapshotId;
  }

  function resolve(sandboxId: string, rel: string): string {
    const base = dirOf(sandboxId);
    const full = path.resolve(base, rel);
    if (full !== base && !full.startsWith(`${base}${path.sep}`))
      throw new FsError('invalid_path', `${rel} is outside the workspace`);
    return full;
  }

  async function entryOf(sandboxId: string, full: string): Promise<Entry | null> {
    const stat = await fs.stat(full).catch(() => null);
    if (!stat) return null;
    const rel = path.relative(dirOf(sandboxId), full) || '.';
    return {
      name: path.basename(full),
      path: rel,
      type: stat.isDirectory() ? 'dir' : 'file',
      size: stat.size,
      mode: stat.mode,
      permissions: (stat.mode & 0o777).toString(8),
      modifiedTime: stat.mtime.toISOString(),
    };
  }

  function running(sandboxId: string | undefined): SandboxRecord {
    const record = sandboxId ? sandboxes.get(sandboxId) : undefined;
    if (!record) throw new Error(`sandbox ${sandboxId} does not exist`);
    if (!record.live)
      throw new Error(
        `sandbox ${sandboxId} is not running (idle-expired or killed): connect to it first`
      );
    record.lastUsed = new Date();
    return record;
  }

  async function command(
    record: SandboxRecord,
    cmd: string,
    timeoutMs: number | undefined,
    signal: AbortSignal
  ) {
    const entry: ProviderLog['commands'][number] = {
      sandboxId: record.sandboxId,
      cmd,
      killed: false,
    };
    log.commands.push(entry);
    const child = spawn('sh', ['-c', cmd.replace(WORKSPACE_IN_COMMAND, dirOf(record.sandboxId))], {
      cwd: dirOf(record.sandboxId),
      detached: true,
    });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (chunk) => (stdout += chunk));
    child.stderr.on('data', (chunk) => (stderr += chunk));
    let timedOut = false;
    // The whole process group, so a `sleep` the shell started dies with it, as in a sandbox.
    const kill = () => {
      entry.killed = true;
      if (child.pid) process.kill(-child.pid, 'SIGKILL');
    };
    const timer = timeoutMs
      ? setTimeout(() => {
          timedOut = true;
          kill();
        }, timeoutMs)
      : undefined;
    signal.addEventListener('abort', kill, { once: true });
    const exitCode = await new Promise<number>((done) =>
      child.on('close', (code) => done(code ?? 137))
    );
    clearTimeout(timer);
    signal.removeEventListener('abort', kill);
    entry.exitCode = timedOut ? 124 : exitCode;
    return { stdout: stdout.trim(), stderr: stderr.trim(), exitCode: entry.exitCode };
  }

  async function files(record: SandboxRecord, op: string, input: Record<string, unknown>) {
    const { sandboxId } = record;
    const target = typeof input.path === 'string' ? resolve(sandboxId, input.path) : '';
    switch (op) {
      case 'read': {
        const entry = await entryOf(sandboxId, target);
        if (!entry) throw new FsError('not_found', `${input.path} does not exist`);
        if (entry.type === 'dir')
          throw new FsError('is_a_directory', `${input.path} is a directory`);
        return { content: (await fs.readFile(target)).toString('base64') };
      }
      case 'write': {
        const entries: Entry[] = [];
        for (const file of input.files as { path: string; content?: string }[]) {
          const full = resolve(sandboxId, file.path);
          await fs.mkdir(path.dirname(full), { recursive: true });
          await fs.writeFile(full, Buffer.from(file.content ?? '', 'base64'));
          entries.push((await entryOf(sandboxId, full))!);
        }
        return { entries, changed: true };
      }
      case 'list': {
        const entry = await entryOf(sandboxId, target);
        if (!entry) throw new FsError('not_found', `${input.path} does not exist`);
        if (entry.type !== 'dir')
          throw new FsError('not_a_directory', `${input.path} is not a directory`);
        const names = await fs.readdir(target);
        const entries = await Promise.all(
          names.map((name) => entryOf(sandboxId, path.join(target, name)))
        );
        return { entries: entries.filter((e) => e !== null) };
      }
      case 'stat':
        return { entry: await entryOf(sandboxId, target) };
      case 'mkdir': {
        const existed = (await entryOf(sandboxId, target)) !== null;
        await fs.mkdir(target, { recursive: true });
        return { created: !existed, changed: !existed };
      }
      case 'remove':
        await fs.rm(target, { recursive: true, force: true });
        return { changed: true };
      case 'rename': {
        const destination = resolve(sandboxId, input.newPath as string);
        await fs.mkdir(path.dirname(destination), { recursive: true });
        await fs.rename(target, destination);
        return { entry: await entryOf(sandboxId, destination), changed: true };
      }
      default:
        throw new Error(`the fake provider does not support $fs.${op}`);
    }
  }

  async function turn(turn: TurnInput, signal: AbortSignal) {
    const { fn, input } = turn;
    switch (fn) {
      case '$create': {
        const sandboxId = turn.sandboxId ?? `sb-${randomUUID().slice(0, 8)}`;
        if (sandboxes.has(sandboxId)) throw new Error(`sandbox ${sandboxId} already exists`);
        await fs.mkdir(dirOf(sandboxId), { recursive: true });
        if (turn.snapshotId)
          await fs.cp(snapshotDir(turn.snapshotId), dirOf(sandboxId), { recursive: true });
        const record: SandboxRecord = {
          sandboxId,
          metadata: turn.metadata ?? {},
          live: true,
          createdAt: new Date(),
        };
        sandboxes.set(sandboxId, record);
        return { sandboxId, output: {}, snapshotId: await snapshot(record) };
      }
      case '$connect': {
        const record = sandboxes.get(turn.sandboxId ?? '');
        if (!record) throw new Error(`sandbox ${turn.sandboxId} does not exist`);
        record.live = true;
        return { sandboxId: record.sandboxId, output: {} };
      }
      case '$list': {
        const filter = input as { sandboxId?: string; metadata?: Record<string, string> };
        const rows = [...sandboxes.values()].filter(
          (s) =>
            (!filter.sandboxId || s.sandboxId === filter.sandboxId) &&
            Object.entries(filter.metadata ?? {}).every(([k, v]) => s.metadata[k] === v)
        );
        return { sandboxId: '', output: { sandboxes: rows } };
      }
      case '$destroy': {
        const record = sandboxes.get(turn.sandboxId ?? '');
        if (record) record.live = false;
        return { sandboxId: turn.sandboxId ?? '', output: {} };
      }
      case '$delete': {
        sandboxes.delete(turn.sandboxId ?? '');
        await fs.rm(dirOf(turn.sandboxId ?? ''), { recursive: true, force: true });
        return { sandboxId: turn.sandboxId ?? '', output: {} };
      }
      case '$command': {
        const record = running(turn.sandboxId);
        const { cmd, timeoutMs } = input as { cmd: string; timeoutMs?: number };
        const output = await command(record, cmd, timeoutMs, signal);
        return { sandboxId: record.sandboxId, output, snapshotId: await snapshot(record) };
      }
      default: {
        if (!fn.startsWith('$fs.')) throw new Error(`the fake provider does not support ${fn}`);
        const record = running(turn.sandboxId);
        try {
          const output = await files(record, fn.slice('$fs.'.length), input);
          const snapshotId = output.changed ? await snapshot(record) : undefined;
          return { sandboxId: record.sandboxId, output, snapshotId };
        } catch (error) {
          if (!(error instanceof FsError)) throw error;
          return {
            sandboxId: record.sandboxId,
            output: { error: { code: error.code, message: error.message } },
          };
        }
      }
    }
  }

  const task = hatchet.task({
    name: 'hatchet-sandbox',
    retries: 0,
    executionTimeout: '10m',
    fn: async (input, ctx) =>
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (await turn(turnInput.parse(input), ctx.abortController.signal)) as any,
  });

  return {
    task,
    log,
    /** The folder that holds a sandbox's workspace. */
    workspaceOf: dirOf,
    workspace: WORKSPACE,
  };
}
