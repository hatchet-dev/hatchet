// A pi-durable ExecutionEnv backed by a Hatchet sandbox. pi's coding tools (read, write, edit, bash) only touch files
// and processes through this object, so the harness can run on the worker while every tool call runs in the sandbox.
// Each method is one or more sandbox calls, which the sandbox provider (brickie) serves as Hatchet tasks.
import * as posix from 'node:path/posix';
import type { Context } from '@earendil-works/chord/context';
import {
  type BinaryReader,
  type DirReader,
  err,
  ExecutionError,
  type ExecutionEnv,
  FileError,
  type FileErrorCode,
  type FileInfo,
  type FileWatcher,
  type LineScan,
  LineScanner,
  ok,
  type ShellExecOptions,
  type ShellExecResult,
  type FileChange,
  type WatchTarget,
} from '@earendil-works/pi-durable/env';
import {
  type CommandResult,
  type EntryInfo,
  FilesystemError,
  type Sandbox,
} from '@hatchet-dev/sandbox';
import type { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';

export const DEFAULT_WORKSPACE = '/workspace';
const DEFAULT_EXEC_TIMEOUT_MS = 10 * 60_000;
const DEFAULT_WATCH_INTERVAL_MS = 500;
/** The task the sandbox provider serves. `exec` starts it directly, to cancel a command whose caller aborts. */
const SANDBOX_TASK = 'brickie-sandbox';
/** The provider trims whitespace around a command's output; this character around each stream keeps it exact. */
const EDGE = '\u001e';
/** What the argv form prints to stderr, with exit code 127, when the program does not exist. */
const NO_PROGRAM = '\u001fpi-durable:no-program';

export type SandboxEnvOptions = {
  /** The image the sandbox runs; `connect` needs it to restart a stopped sandbox. */
  template: string;
  /** The sandbox's workspace directory. Paths outside it are refused. Default `/workspace`. */
  workspace?: string;
  /** The working directory tools start in. Default: the workspace. */
  cwd?: string;
  /** How long a command may run when the tool sets no timeout. Default 10 minutes. */
  execTimeoutMs?: number;
  /**
   * How often a watcher compares the watched files. Each comparison is one sandbox command, which also makes a
   * snapshot. Default 500 ms.
   */
  watchIntervalMs?: number;
};

const ERROR_CODES: Partial<Record<FilesystemError['code'], FileErrorCode>> = {
  not_found: 'not_found',
  is_a_directory: 'is_directory',
  not_a_directory: 'not_directory',
  invalid_path: 'invalid',
  invalid: 'invalid',
};

const handles = new WeakMap<HatchetClient, Map<string, Promise<Sandbox>>>();

/** One attached handle per sandbox per client: `connect` restarts a stopped sandbox from its latest snapshot. */
export function sandboxHandle(
  hatchet: HatchetClient,
  sandboxId: string,
  template: string
): Promise<Sandbox> {
  let byId = handles.get(hatchet);
  if (!byId) handles.set(hatchet, (byId = new Map()));
  let handle = byId.get(sandboxId);
  if (!handle) {
    handle = hatchet.sandbox.connect(sandboxId, { template });
    handle.catch(() => byId.delete(sandboxId));
    byId.set(sandboxId, handle);
  }
  return handle;
}

/** The snapshot holding the sandbox's current workspace. Every exec and write makes one. */
export async function latestSnapshot(
  hatchet: HatchetClient,
  sandboxId: string
): Promise<string | undefined> {
  return (await hatchet.sandbox.list({ sandboxId }))[0]?.latestSnapshot;
}

function quote(value: string): string {
  return `'${value.replaceAll("'", `'\\''`)}'`;
}

function toFileError(error: unknown, path: string): FileError {
  if (error instanceof FilesystemError)
    return new FileError(ERROR_CODES[error.code] ?? 'unknown', error.message, path);
  return new FileError('unknown', error instanceof Error ? error.message : String(error), path);
}

const aborted = (context: Context) => context.abortSignal?.aborted === true;
const abortedFile = (path?: string) =>
  err<never, FileError>(new FileError('aborted', 'The operation was aborted', path));
const invalid = (message: string, path?: string) =>
  err<never, FileError>(new FileError('invalid', message, path));
const isCount = (value: number) => Number.isInteger(value) && value >= 0;

/** Strips the edge markers `exec` puts around a stream. */
function unwrap(text: string): string {
  const start = text.startsWith(EDGE) ? 1 : 0;
  const end = text.endsWith(EDGE) && text.length > start ? text.length - 1 : text.length;
  return text.slice(start, end);
}

function notSupported(name: string) {
  return Promise.resolve(
    err<never, FileError>(new FileError('not_supported', `${name} is not supported in the sandbox`))
  );
}

export class SandboxEnv implements ExecutionEnv {
  readonly id: string;
  readonly template: string;
  readonly workspace: string;
  cwd: string;
  private readonly execTimeoutMs: number;
  private readonly watchIntervalMs: number;

  constructor(
    private readonly hatchet: HatchetClient,
    readonly sandboxId: string,
    options: SandboxEnvOptions
  ) {
    this.id = `hatchet-sandbox:${sandboxId}`;
    this.template = options.template;
    this.workspace = options.workspace ?? DEFAULT_WORKSPACE;
    this.cwd = options.cwd ?? this.workspace;
    this.execTimeoutMs = options.execTimeoutMs ?? DEFAULT_EXEC_TIMEOUT_MS;
    this.watchIntervalMs = options.watchIntervalMs ?? DEFAULT_WATCH_INTERVAL_MS;
  }

  /** The sandbox's latest snapshot, for recording after a call that changed the workspace. */
  latestSnapshot(): Promise<string | undefined> {
    return latestSnapshot(this.hatchet, this.sandboxId);
  }

  private sandbox(): Promise<Sandbox> {
    return sandboxHandle(this.hatchet, this.sandboxId, this.template);
  }

  /** The sandbox's file API takes workspace-relative paths; anything outside the workspace is refused. */
  private relative(path: string): string | FileError {
    const absolute = posix.resolve(this.cwd, path);
    const rel = posix.relative(this.workspace, absolute);
    if (rel.startsWith('..'))
      return new FileError('permission_denied', `${absolute} is outside ${this.workspace}`, path);
    return rel || '.';
  }

  /** The absolute path of a workspace path, or the error for a path outside the workspace. */
  private inside(path: string): string | FileError {
    const rel = this.relative(path);
    return rel instanceof FileError ? rel : posix.join(this.workspace, rel);
  }

  private async files<T>(path: string, fn: (sandbox: Sandbox, rel: string) => Promise<T>) {
    const rel = this.relative(path);
    if (rel instanceof FileError) return err<T, FileError>(rel);
    try {
      return ok<T, FileError>(await fn(await this.sandbox(), rel));
    } catch (error) {
      return err<T, FileError>(toFileError(error, path));
    }
  }

  /** Runs a script in the sandbox for the env's own checks; the output is exact, as in `exec`. */
  private async sh(script: string): Promise<CommandResult> {
    const sandbox = await this.sandbox();
    const result = await sandbox.exec(this.wrap(script));
    return { ...result, stdout: unwrap(result.stdout ?? ''), stderr: unwrap(result.stderr ?? '') };
  }

  private toFileInfo(info: EntryInfo): FileInfo {
    return {
      name: info.name,
      path: posix.join(this.workspace, info.path),
      kind: info.type === 'dir' ? 'directory' : info.type,
      size: info.size,
      mtimeMs: info.modifiedTime?.getTime() ?? 0,
    };
  }

  async absolutePath(path: string, _context: Context) {
    return ok<string, FileError>(posix.resolve(this.cwd, path));
  }

  async joinPath(parts: string[], _context: Context) {
    return ok<string, FileError>(posix.join(...parts));
  }

  async canonicalPath(path: string, _context: Context) {
    return ok<string, FileError>(posix.resolve(this.cwd, path));
  }

  exists(path: string, _context: Context) {
    return this.files(path, (sandbox, rel) => sandbox.files.exists(rel));
  }

  fileInfo(path: string, _context: Context) {
    return this.files(path, async (sandbox, rel) =>
      this.toFileInfo(await sandbox.files.getInfo(rel))
    );
  }

  listDir(path: string, _context: Context) {
    return this.files(path, async (sandbox, rel) =>
      (await sandbox.files.list(rel)).map((info) => this.toFileInfo(info))
    );
  }

  readTextFile(path: string, _context: Context) {
    return this.files(path, (sandbox, rel) => sandbox.files.read(rel));
  }

  readBinaryFile(path: string, _context: Context) {
    return this.files(path, (sandbox, rel) => sandbox.files.read(rel, { format: 'bytes' }));
  }

  async readTextLines(path: string, options: { maxLines?: number } | undefined, context: Context) {
    const text = await this.readTextFile(path, context);
    if (!text.ok) return text;
    const lines = text.value.split('\n');
    return ok<string[], FileError>(
      options?.maxLines === undefined ? lines : lines.slice(0, options.maxLines)
    );
  }

  /**
   * `read` opens files through this. The sandbox has no positional reads, so the reader holds the whole file, read when
   * it opens: it keeps reading that file after a rename.
   */
  async openBinaryReader(
    path: string,
    options: { noFollow?: boolean } | undefined,
    context: Context
  ) {
    if (aborted(context)) return abortedFile(path);
    if (options?.noFollow) {
      // The sandbox's file calls follow symbolic links, so only a shell can tell a link apart.
      const absolute = this.inside(path);
      if (absolute instanceof FileError) return err<BinaryReader, FileError>(absolute);
      const link = await this.sh(`[ -L ${quote(absolute)} ]`);
      if (link.exitCode === 0) return invalid(`${absolute} is a symbolic link`, path);
    }
    const [bytes, info] = await Promise.all([
      this.readBinaryFile(path, context),
      this.fileInfo(path, context),
    ]);
    if (!info.ok) return info;
    if (info.value.kind === 'directory')
      return err<BinaryReader, FileError>(
        new FileError('is_directory', `${path} is a directory`, path)
      );
    if (info.value.kind !== 'file') return invalid(`${path} is not a regular file`, path);
    if (!bytes.ok) return bytes;
    const data = bytes.value;
    let closed = false;
    const usable = (context: Context) =>
      closed
        ? invalid('The reader is closed', path)
        : aborted(context)
          ? abortedFile(path)
          : undefined;
    return ok<BinaryReader, FileError>({
      info: async (context) => usable(context) ?? ok<FileInfo, FileError>(info.value),
      read: async (offset, length, context) => {
        const refused = usable(context);
        if (refused) return refused;
        if (!isCount(offset) || !isCount(length))
          return invalid(`Invalid range: ${offset}, ${length}`, path);
        return ok<Uint8Array, FileError>(data.slice(offset, offset + length));
      },
      scanLines: async (range, context) => {
        const refused = usable(context);
        if (refused) return refused;
        let scanner: LineScanner;
        try {
          scanner = new LineScanner(range.startLine, range.endLine);
        } catch (error) {
          return invalid(error instanceof Error ? error.message : String(error), path);
        }
        scanner.push(data);
        return ok<LineScan, FileError>(scanner.finish());
      },
      close: async () => {
        closed = true;
      },
    });
  }

  writeFile(path: string, content: string | Uint8Array, _context: Context) {
    return this.files(path, async (sandbox, rel) => {
      await sandbox.files.write(rel, content);
    });
  }

  renameFile(sourcePath: string, destinationPath: string, _context: Context) {
    const destination = this.relative(destinationPath);
    if (destination instanceof FileError) return Promise.resolve(err<void, FileError>(destination));
    return this.files(sourcePath, async (sandbox, rel) => {
      await sandbox.files.rename(rel, destination);
    });
  }

  createDir(path: string, _options: { recursive?: boolean } | undefined, _context: Context) {
    return this.files(path, async (sandbox, rel) => {
      await sandbox.files.makeDir(rel);
    });
  }

  /** The sandbox removes a missing path without an error, so without `force` a missing path is checked first. */
  remove(
    path: string,
    options: { recursive?: boolean; force?: boolean } | undefined,
    _context: Context
  ) {
    return this.files(path, async (sandbox, rel) => {
      if (!options?.force && !(await sandbox.files.exists(rel))) {
        throw new FilesystemError('not_found', `${path} does not exist`);
      }
      await sandbox.files.remove(rel);
    });
  }

  /**
   * Lists the directory when it opens, then pages through those names. Each page lists the directory again, so an entry
   * removed in between is skipped and every entry has its current metadata.
   */
  async openDirReader(path: string, context: Context) {
    if (aborted(context)) return abortedFile(path);
    const listing = await this.listDir(path, context);
    if (!listing.ok) return listing;
    const remaining = listing.value.map((entry) => entry.name);
    let closed = false;
    return ok<DirReader, FileError>({
      next: async (maxEntries, context) => {
        if (closed) return invalid('The reader is closed', path);
        if (!Number.isInteger(maxEntries) || maxEntries < 1)
          return invalid(`Invalid maxEntries: ${maxEntries}`, path);
        if (aborted(context)) return abortedFile(path);
        if (remaining.length === 0) return ok({ entries: [], done: true });
        const now = await this.listDir(path, context);
        if (!now.ok && now.error.code !== 'not_found') return now;
        const current = new Map((now.ok ? now.value : []).map((entry) => [entry.name, entry]));
        const entries: FileInfo[] = [];
        while (remaining.length > 0 && entries.length < maxEntries) {
          const entry = current.get(remaining.shift()!);
          if (entry) entries.push(entry);
        }
        return ok({ entries, done: remaining.length === 0 });
      },
      close: async () => {
        closed = true;
      },
    });
  }

  /**
   * Polls: every `watchIntervalMs`, one command stats every watched path, and the paths whose type, inode, size or
   * times differ from the previous poll are reported. Two writes of the same size within one second can look alike.
   */
  async watch(
    targets: readonly WatchTarget[],
    onChange: (change: FileChange) => void,
    context: Context
  ) {
    if (aborted(context)) return abortedFile();
    const script: string[] = [];
    const fmt = quote('%F|%i|%s|%Y|%Z|%n');
    for (const target of targets) {
      const absolute = this.inside(target.path);
      if (absolute instanceof FileError) return err<FileWatcher, FileError>(absolute);
      const names = [...(target.exclude?.hidden ? ['.*'] : []), ...(target.exclude?.names ?? [])];
      const prune =
        names.length === 0
          ? ''
          : `\\( ${names.map((name) => `-name ${quote(name)}`).join(' -o ')} \\) -prune -o`;
      const depth = target.recursive ? '' : '-maxdepth 1';
      // The target itself follows a symbolic link; entries below it are listed, not followed.
      script.push(`stat -L -c ${fmt} ${quote(absolute)} 2>/dev/null`);
      script.push(
        `[ -d ${quote(absolute)} ] && find ${quote(`${absolute}/`)} -mindepth 1 ${depth} ${prune} -exec stat -c ${fmt} {} + 2>/dev/null`
      );
    }
    const poll = async (): Promise<Map<string, string>> => {
      const result = await this.sh(`${script.join('\n')}\ntrue`);
      if (result.exitCode !== 0)
        throw new Error(`the watch command exited with ${result.exitCode}: ${result.stderr}`);
      const seen = new Map<string, string>();
      for (const line of result.stdout.split('\n')) {
        const fields = line.split('|');
        if (fields.length < 6) continue;
        seen.set(
          posix.normalize(fields.slice(5).join('|')).replace(/(.)\/$/, '$1'),
          fields.slice(0, 5).join('|')
        );
      }
      return seen;
    };

    let previous: Map<string, string>;
    try {
      previous = await poll();
    } catch (error) {
      return err<FileWatcher, FileError>(toFileError(error, targets[0]?.path ?? ''));
    }
    let closed = false;
    let uncertain = false;
    let timer: NodeJS.Timeout | undefined;
    let running: Promise<void> = Promise.resolve();
    const report = (change: FileChange) => {
      if (closed) return;
      try {
        onChange(change);
      } catch {
        // the host's callback failing must not stop the watcher
      }
    };
    const tick = () => {
      running = (async () => {
        try {
          const current = await poll();
          if (closed) return;
          if (uncertain) {
            uncertain = false;
            report({ kind: 'overflow' });
          }
          const paths = new Set<string>();
          for (const [path, sig] of current) if (previous.get(path) !== sig) paths.add(path);
          for (const path of previous.keys()) if (!current.has(path)) paths.add(path);
          previous = current;
          if (paths.size > 0) report({ kind: 'paths', paths: [...paths] });
        } catch {
          uncertain = true; // a failed poll may have missed a change: the next good one reports an overflow
        }
        if (!closed) timer = setTimeout(tick, this.watchIntervalMs);
      })();
    };
    timer = setTimeout(tick, this.watchIntervalMs);
    return ok<FileWatcher, FileError>({
      mode: 'polling',
      stop: async () => {
        closed = true;
        clearTimeout(timer);
        await running;
      },
    });
  }

  openTextLineReader() {
    return notSupported('openTextLineReader');
  }
  appendFile() {
    return notSupported('appendFile');
  }
  truncateFile() {
    return notSupported('truncateFile');
  }
  flushFile() {
    return notSupported('flushFile');
  }
  createTempDir() {
    return notSupported('createTempDir');
  }
  createTempFile() {
    return notSupported('createTempFile');
  }

  /**
   * Wraps a script so its output comes back exact: an edge marker around each stream, which the provider's trimming
   * cannot touch. The script runs in a subshell, so an `exit` in it still prints the closing markers.
   */
  private wrap(script: string): string {
    const edge = `printf '\\036'; printf '\\036' >&2`;
    return `${edge}\n( ${script}\n)\n__pi_status=$?\n${edge}\nexit $__pi_status`;
  }

  /**
   * Runs through the sandbox's `sh -c`. The provider returns the output once the command ends, so `onOutput` gets it
   * in one chunk per stream rather than as it arrives. An abort cancels the command's sandbox run, which kills it.
   */
  async exec(
    command: string | readonly string[],
    options: ShellExecOptions | undefined,
    context: Context
  ) {
    const failed = (code: ExecutionError['code'], message: string) =>
      err<ShellExecResult, ExecutionError>(new ExecutionError(code, message));
    if (typeof command !== 'string' && command.length === 0)
      return failed('spawn_error', 'No program to run');
    if (aborted(context)) return failed('aborted', 'The command was aborted');

    // The argv form runs its program directly; a program that does not exist is a spawn error, as in a local process.
    const script =
      typeof command === 'string'
        ? command
        : `command -v ${quote(command[0]!)} >/dev/null 2>&1 || { printf '%s' ${quote(NO_PROGRAM)} >&2; exit 127; }\nexec ${command.map(quote).join(' ')}`;
    const setup = [
      `cd ${quote(posix.resolve(this.cwd, options?.cwd ?? '.'))}`,
      ...Object.entries(options?.env ?? {}).map(([key, value]) => `export ${key}=${quote(value)}`),
    ].join(' && ');
    const timeoutMs = options?.timeout === undefined ? this.execTimeoutMs : options.timeout * 1000;

    let result: CommandResult;
    try {
      await this.sandbox(); // restarts a stopped sandbox from its snapshot
      const run = await this.hatchet.workflow({ name: SANDBOX_TASK }).runNoWait({
        sandboxId: this.sandboxId,
        image: this.template,
        fn: '$command',
        input: { cmd: this.wrap(`${setup} && { ${script}\n}`), timeoutMs },
      });
      const output = (async () => {
        const raw = (await run.output) as Record<string, unknown>;
        const turn = (raw[SANDBOX_TASK] ?? raw) as { output?: CommandResult };
        if (typeof turn.output?.exitCode !== 'number')
          throw new Error('The sandbox run ended without a result');
        return turn.output;
      })();
      const signal = context.abortSignal;
      if (!signal) result = await output;
      else {
        const abort = new Promise<'aborted'>((resolve) =>
          signal.addEventListener('abort', () => resolve('aborted'), { once: true })
        );
        const first = await Promise.race([output, abort]);
        if (first === 'aborted') {
          output.catch(() => {});
          // Cancelling the run makes the provider kill the command. If that fails, the command runs to its end.
          void run
            .getWorkflowRunId()
            .then((id) => this.hatchet.runs.cancel({ ids: [id] }))
            .catch((error) =>
              console.warn(`${this.id}: could not cancel an aborted command: ${String(error)}`)
            );
          return failed('aborted', 'The command was aborted');
        }
        result = first;
      }
    } catch (error) {
      return failed('unknown', error instanceof Error ? error.message : String(error));
    }

    const stdout = unwrap(result.stdout ?? '');
    let stderr = unwrap(result.stderr ?? '');
    if (typeof command !== 'string' && result.exitCode === 127 && stderr.endsWith(NO_PROGRAM)) {
      return failed('spawn_error', `${command[0]} was not found`);
    }
    if (stderr.endsWith(NO_PROGRAM)) stderr = stderr.slice(0, -NO_PROGRAM.length);
    if (stdout) options?.onOutput?.(stdout, context, { stream: 'stdout' });
    if (stderr) options?.onOutput?.(stderr, context, { stream: 'stderr' });
    // the provider reports a timeout as exit code 124
    if (result.exitCode === 124 && options?.timeout !== undefined) {
      return failed('timeout', `Timed out after ${options.timeout}s`);
    }
    return ok<ShellExecResult, ExecutionError>({ exitCode: result.exitCode });
  }

  async cleanup(_context: Context) {}
}
