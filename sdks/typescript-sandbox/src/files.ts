import * as fs from 'node:fs/promises';
import type { JsonObject } from '@hatchet-dev/typescript-sdk/v1';
import { z } from 'zod';

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

const path = z.string().min(1);
const positiveInt = z.int().positive();
const httpsUrl = z.url({ protocol: /^https$/ });
const fileData = z.union([
  z.string(),
  z.instanceof(Uint8Array),
  z.instanceof(ArrayBuffer),
  z.instanceof(Blob),
]);
const writeEntries = z.strictObject({ path, data: fileData }).array();
const readOpts = z.strictObject({ format: z.enum(['text', 'bytes', 'blob']).optional() });
const listOpts = z.strictObject({ depth: positiveInt.optional() });
const transferOpts = z.strictObject({ timeoutMs: positiveInt.optional() });

const entry = z.looseObject({
  name: z.string(),
  path: z.string(),
  type: z.enum(['file', 'dir']),
  size: z.number(),
  mode: z.number(),
  permissions: z.string(),
  modifiedTime: z.coerce.date().optional(),
});

const fsOutput = z.looseObject({
  content: z.string().optional(),
  stagedKey: z.string().optional(),
  url: z.string().optional(),
  putUrl: z.string().optional(),
  getUrl: z.string().optional(),
  entry: entry.nullish(),
  entries: entry.array().optional(),
  created: z.boolean().optional(),
  error: z.object({ code: z.string(), message: z.string() }).optional(),
});
type FsOutput = z.infer<typeof fsOutput>;
const fsOutputWithEntry = fsOutput.extend({ entry });

/** @internal */
export type FsTurn = (fn: string, input: JsonObject, fresh?: boolean) => Promise<JsonObject>;

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

  private call(fn: string, input: JsonObject, fresh?: boolean): Promise<FsOutput>;
  private call<S extends z.ZodType<FsOutput>>(
    fn: string,
    input: JsonObject,
    fresh: boolean,
    schema: S
  ): Promise<z.infer<S>>;
  private async call(
    fn: string,
    input: JsonObject,
    fresh = false,
    schema: z.ZodType<FsOutput> = fsOutput
  ): Promise<FsOutput> {
    const out = fsOutput.parse(await this.turn(`$fs.${fn}`, input, fresh));
    if (out.error)
      throw new FilesystemError(out.error.code as FilesystemErrorCode, out.error.message);
    return schema.parse(out);
  }

  /** Reads a file as text (default), bytes, or a Blob. Files over the provider's `--max-file-bytes` (2 MiB) come through the provider's `--store` instead of Hatchet. */
  read(path: string, opts?: ReadOpts & { format?: 'text' }): Promise<string>;
  read(path: string, opts: ReadOpts & { format: 'bytes' }): Promise<Uint8Array>;
  read(path: string, opts: ReadOpts & { format: 'blob' }): Promise<Blob>;
  async read(p: string, opts: ReadOpts = {}): Promise<string | Uint8Array | Blob> {
    const { format } = readOpts.parse(opts);
    const out = await this.call('read', { path: path.parse(p) });
    const bytes = out.url
      ? await this.readStaged(out.url, out.stagedKey!)
      : Buffer.from(out.content ?? '', 'base64');
    if (format === 'bytes') return new Uint8Array(bytes);
    if (format === 'blob') return new Blob([new Uint8Array(bytes)]);
    return bytes.toString('utf8');
  }

  /** A replayed durable read may hold a URL that has expired; the provider signs the same staged file again. The renewal is a plain run, not a durable child, so the replayed child sequence stays aligned. */
  private async readStaged(url: string, stagedKey: string): Promise<Buffer> {
    try {
      return await getStaged(url);
    } catch (e) {
      if (url.startsWith('file://') || !(e instanceof FilesystemError)) throw e;
      const { getUrl } = await this.call('presign', { stagedKey }, true);
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
    writeEntries.parse(files);
    const bytes = await Promise.all(files.map((f) => toBytes(f.data)));
    const encoded: Array<{ path: string; content?: string; stagedKey?: string }> = [];
    for (const [i, f] of files.entries()) {
      const b = bytes[i];
      if (b.length <= STAGE_OVER_BYTES) {
        encoded.push({ path: f.path, content: b.toString('base64') });
        continue;
      }
      const { stagedKey, putUrl } = await this.call('presign', {});
      try {
        await putStaged(putUrl!, b);
        encoded.push({ path: f.path, stagedKey: stagedKey! });
      } catch {
        const again = await this.call('presign', {}, true);
        await putStaged(again.putUrl!, b);
        encoded.push({ path: f.path, stagedKey: again.stagedKey! });
      }
    }
    const { entries = [] } = await this.call('write', { files: encoded });
    return entries;
  }

  /** Lists a directory. `depth` 1 (default) is the directory itself; higher values recurse. No snapshot is made. */
  async list(p = '.', opts: ListOpts = {}): Promise<EntryInfo[]> {
    const { depth = 1 } = listOpts.parse(opts);
    const { entries = [] } = await this.call('list', { path: path.parse(p), depth });
    return entries;
  }

  /** True if the path exists. No snapshot is made. */
  async exists(p: string): Promise<boolean> {
    const { entry } = await this.call('stat', { path: path.parse(p) });
    return entry != null;
  }

  /** Metadata of a file or directory. Throws `FilesystemError` with code `not_found` if absent. */
  async getInfo(p: string): Promise<EntryInfo> {
    const { entry } = await this.call('stat', { path: path.parse(p) });
    if (!entry) throw new FilesystemError('not_found', `${p} does not exist`);
    return entry;
  }

  /** Creates a directory and its parents. Returns false if it already existed. */
  async makeDir(p: string): Promise<boolean> {
    const { created } = await this.call('mkdir', { path: path.parse(p) });
    return created === true;
  }

  /** Renames or moves a file or directory. */
  async rename(oldPath: string, newPath: string): Promise<EntryInfo> {
    const { entry } = await this.call(
      'rename',
      { path: path.parse(oldPath), newPath: path.parse(newPath) },
      false,
      fsOutputWithEntry
    );
    return entry;
  }

  /** Copies a file or directory tree. */
  async copy(srcPath: string, dstPath: string): Promise<EntryInfo> {
    const { entry } = await this.call(
      'copy',
      { path: path.parse(srcPath), newPath: path.parse(dstPath) },
      false,
      fsOutputWithEntry
    );
    return entry;
  }

  /** Deletes a file or directory tree. */
  async remove(p: string): Promise<void> {
    await this.call('remove', { path: path.parse(p) });
  }
  /** The provider fetches an https URL into the workspace. The sandbox needs no network. Private and loopback addresses are refused. */
  async download(url: string, p: string, opts: TransferOpts = {}): Promise<EntryInfo> {
    const { entry } = await this.call(
      'download',
      { url: httpsUrl.parse(url), path: path.parse(p), ...transferOpts.parse(opts) },
      false,
      fsOutputWithEntry
    );
    return entry;
  }
  /** The provider sends a workspace file to an https URL with HTTP PUT, for example a presigned URL. */
  async upload(p: string, url: string, opts: TransferOpts = {}): Promise<void> {
    await this.call('upload', {
      path: path.parse(p),
      url: httpsUrl.parse(url),
      ...transferOpts.parse(opts),
    });
  }
}
