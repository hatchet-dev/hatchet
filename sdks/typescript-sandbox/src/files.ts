import * as fs from 'node:fs/promises';
import type { JsonObject } from '@hatchet-dev/typescript-sdk/v1';

export type FileType = 'file' | 'dir';

export interface EntryInfo {
  name: string;
  path: string;
  type: FileType;
  size: number;
  mode: number;
  permissions: string;
  modifiedTime?: Date;
}

export type FileData = string | ArrayBuffer | Uint8Array | Blob;

export interface WriteEntry {
  path: string;
  data: FileData;
}

export interface ReadOpts {
  format?: 'text' | 'bytes' | 'blob';
}

export interface TransferOpts {
  /** Give up after this long. */
  timeoutMs?: number;
}

export interface ListOpts {
  depth?: number;
}

export type FilesystemErrorCode =
  | 'staging_failed'
  | 'not_found'
  | 'invalid_path'
  | 'invalid_url'
  | 'invalid_content'
  | 'invalid'
  | 'is_a_directory'
  | 'not_a_directory'
  | 'too_large'
  | 'transfer_failed';

/** Thrown for caller mistakes such as a missing file or a path outside the workspace. */
export class FilesystemError extends Error {
  constructor(
    readonly code: FilesystemErrorCode,
    message: string
  ) {
    super(message);
    this.name = 'FilesystemError';
  }
}

type RawEntry = Omit<EntryInfo, 'modifiedTime'> & { modifiedTime?: string };

interface FsOutput extends JsonObject {
  content?: string;
  stagedKey?: string;
  url?: string;
  putUrl?: string;
  getUrl?: string;
  entry?: RawEntry | null;
  entries?: RawEntry[];
  created?: boolean;
  error?: { code: FilesystemErrorCode; message: string };
}

/** @internal */
export type FsTurn = (fn: string, input: JsonObject) => Promise<FsOutput>;

const toEntry = (e: RawEntry): EntryInfo => ({
  ...e,
  modifiedTime: e.modifiedTime ? new Date(e.modifiedTime) : undefined,
});

const STAGE_OVER_BYTES = 2 << 20;

async function toBytes(data: FileData): Promise<Buffer> {
  if (typeof data === 'string') return Buffer.from(data, 'utf8');
  if (data instanceof Blob) return Buffer.from(await data.arrayBuffer());
  if (data instanceof Uint8Array) return Buffer.from(data.buffer, data.byteOffset, data.byteLength);
  return Buffer.from(data);
}

async function putStaged(url: string, bytes: Buffer): Promise<void> {
  if (url.startsWith('file://')) {
    await fs.writeFile(decodeURIComponent(url.slice('file://'.length)), bytes);
    return;
  }
  const res = await fetch(url, {
    method: 'PUT',
    body: new Uint8Array(bytes),
    headers: { 'Content-Type': 'application/octet-stream' },
  });
  if (!res.ok)
    throw new FilesystemError('staging_failed', `stage upload failed: HTTP ${res.status}`);
}

async function getStaged(url: string): Promise<Buffer> {
  if (url.startsWith('file://')) {
    return fs.readFile(decodeURIComponent(url.slice('file://'.length)));
  }
  const res = await fetch(url);
  if (!res.ok)
    throw new FilesystemError('staging_failed', `stage download failed: HTTP ${res.status}`);
  return Buffer.from(await res.arrayBuffer());
}
/** File operations on a sandbox workspace. Paths are relative to the workspace. Each call is one Hatchet task. */
export class Files {
  /** @internal */
  constructor(private readonly turn: FsTurn) {}

  private async call(fn: string, input: JsonObject): Promise<FsOutput> {
    const out = await this.turn(`$fs.${fn}`, input);
    if (out.error) throw new FilesystemError(out.error.code, out.error.message);
    return out;
  }

  /** Reads a file as text (default), bytes, or a Blob. Files over the provider's `--max-file-bytes` (2 MiB) come through the provider's `--store` instead of Hatchet. */
  read(path: string, opts?: ReadOpts & { format?: 'text' }): Promise<string>;
  read(path: string, opts: ReadOpts & { format: 'bytes' }): Promise<Uint8Array>;
  read(path: string, opts: ReadOpts & { format: 'blob' }): Promise<Blob>;
  async read(path: string, opts: ReadOpts = {}): Promise<string | Uint8Array | Blob> {
    const out = await this.call('read', { path });
    const bytes = out.url
      ? await this.readStaged(out.url, out.stagedKey!)
      : Buffer.from(out.content ?? '', 'base64');
    if (opts.format === 'bytes') return new Uint8Array(bytes);
    if (opts.format === 'blob') return new Blob([new Uint8Array(bytes)]);
    return bytes.toString('utf8');
  }

  /** A replayed durable read may hold a URL that has expired; the provider signs the same staged file again. */
  private async readStaged(url: string, stagedKey: string): Promise<Buffer> {
    try {
      return await getStaged(url);
    } catch (e) {
      if (url.startsWith('file://') || !(e instanceof FilesystemError)) throw e;
      const { getUrl } = await this.call('presign', { stagedKey });
      return getStaged(getUrl!);
    }
  }

  /** Writes one file or several. Parent directories are created. Bytes are stored exactly. Files over 2 MiB go through the provider's `--store` instead of Hatchet. */
  write(path: string, data: FileData): Promise<EntryInfo>;
  write(files: WriteEntry[]): Promise<EntryInfo[]>;
  async write(
    pathOrFiles: string | WriteEntry[],
    data?: FileData
  ): Promise<EntryInfo | EntryInfo[]> {
    if (typeof pathOrFiles !== 'string') return this.writeFiles(pathOrFiles);
    const [entry] = await this.writeFiles([{ path: pathOrFiles, data: data ?? '' }]);
    return entry;
  }

  /** Writes several files in one task. */
  async writeFiles(files: WriteEntry[]): Promise<EntryInfo[]> {
    const encoded = await Promise.all(
      files.map(async (f) => {
        const bytes = await toBytes(f.data);
        if (bytes.length <= STAGE_OVER_BYTES)
          return { path: f.path, content: bytes.toString('base64') };
        const { stagedKey, putUrl } = await this.call('presign', {});
        await putStaged(putUrl!, bytes);
        return { path: f.path, stagedKey: stagedKey! };
      })
    );
    const { entries = [] } = await this.call('write', { files: encoded });
    return entries.map(toEntry);
  }

  /** Lists a directory. `depth` 1 (default) is the directory itself; higher values recurse. No snapshot is made. */
  async list(path = '.', opts: ListOpts = {}): Promise<EntryInfo[]> {
    const { entries = [] } = await this.call('list', { path, depth: opts.depth ?? 1 });
    return entries.map(toEntry);
  }

  /** True if the path exists. No snapshot is made. */
  async exists(path: string): Promise<boolean> {
    const { entry } = await this.call('stat', { path });
    return entry != null;
  }

  /** Metadata of a file or directory. Throws `FilesystemError` with code `not_found` if absent. */
  async getInfo(path: string): Promise<EntryInfo> {
    const { entry } = await this.call('stat', { path });
    if (!entry) throw new FilesystemError('not_found', `${path} does not exist`);
    return toEntry(entry);
  }

  /** Creates a directory and its parents. Returns false if it already existed. */
  async makeDir(path: string): Promise<boolean> {
    const { created } = await this.call('mkdir', { path });
    return created === true;
  }

  /** Renames or moves a file or directory. */
  async rename(oldPath: string, newPath: string): Promise<EntryInfo> {
    const { entry } = await this.call('rename', { path: oldPath, newPath });
    return toEntry(entry!);
  }

  /** Copies a file or directory tree. */
  async copy(srcPath: string, dstPath: string): Promise<EntryInfo> {
    const { entry } = await this.call('copy', { path: srcPath, newPath: dstPath });
    return toEntry(entry!);
  }

  /** Deletes a file or directory tree. */
  async remove(path: string): Promise<void> {
    await this.call('remove', { path });
  }
  /** The provider fetches an https URL into the workspace. The sandbox needs no network. Private and loopback addresses are refused. */
  async download(url: string, path: string, opts: TransferOpts = {}): Promise<EntryInfo> {
    const { entry } = await this.call('download', {
      url,
      path,
      ...(opts.timeoutMs ? { timeoutMs: opts.timeoutMs } : {}),
    });
    return toEntry(entry!);
  }
  /** The provider sends a workspace file to an https URL with HTTP PUT, for example a presigned URL. */
  async upload(path: string, url: string, opts: TransferOpts = {}): Promise<void> {
    await this.call('upload', {
      path,
      url,
      ...(opts.timeoutMs ? { timeoutMs: opts.timeoutMs } : {}),
    });
  }
}
