// A pi-durable Storage kept in a Hatchet durable stream topic, so any worker can reopen a session.
//
// The topic is the log and memory is the index, as in pi's JSONL backend: a commit is applied to a MemoryStorage
// and published as one message; opening reads the topic from its first message and applies every commit again.
//
// One owner at a time is assumed, not enforced: a worker that lost its heartbeat but still runs can append next to
// its successor, and the next open then fails on the out-of-order seqs.
import type { Context } from '@earendil-works/chord/context';
import type {
  CommitResult,
  ConversationId,
  ConversationQuery,
  Cursor,
  DocumentAddress,
  DocumentId,
  DocumentPoint,
  DocumentQuery,
  EntryId,
  EntryQuery,
  Id,
  Seq,
  Storage,
  StorageWrite,
  SubmissionId,
  SubmissionQuery,
  TaskId,
  TaskQuery,
} from '@earendil-works/pi-durable';
import { MemoryStorage } from '@earendil-works/pi-durable/storage/memory';
import type { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';

const FORMAT = 1;
const GRPC_NOT_FOUND = 5;

type Message =
  | { format: typeof FORMAT; type: 'header' }
  | { format: typeof FORMAT; type: 'commit'; commits: { seq: number; writes: StorageWrite[] }[] };

export type HatchetStorageOptions = {
  /** The stream namespace of the topic; the empty namespace when omitted. */
  namespace?: string;
};

export class HatchetStorageError extends Error {
  constructor(message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = 'HatchetStorageError';
  }
}

/** Opens the storage kept in `topic`, or starts it when nothing was ever published there. */
export function openHatchetStorage(
  hatchet: HatchetClient,
  topic: string,
  options?: HatchetStorageOptions
): Promise<HatchetStorage> {
  return HatchetStorage.open(hatchet, topic, options);
}

export class HatchetStorage implements Storage {
  private readonly memory = new MemoryStorage();
  private closed = false;
  private poisonError: Error | undefined;
  /** Settles once the commit being published does; memory has its batches already. */
  private writing: Promise<void> | undefined;

  private constructor(
    private readonly hatchet: HatchetClient,
    private readonly topic: string,
    private readonly namespace: string
  ) {}

  static async open(
    hatchet: HatchetClient,
    topic: string,
    options: HatchetStorageOptions = {}
  ): Promise<HatchetStorage> {
    const storage = new HatchetStorage(hatchet, topic, options.namespace ?? '');
    await storage.recover();
    return storage;
  }

  private async recover(): Promise<void> {
    const { streams } = this.hatchet;
    let latestCursor: string | undefined;
    try {
      const metadata = await streams.topicMetadata(this.topic, { namespace: this.namespace });
      // The topic exists but retains nothing: every message, the header included, is past retention.
      if (metadata.latestCursor === undefined) throw this.expired();
      latestCursor = String(metadata.latestCursor);
    } catch (error) {
      if (errorCode(error) !== GRPC_NOT_FOUND) throw error;
    }

    // Only NOT_FOUND starts a new storage: any other failure must not pass for an empty session. A topic the engine
    // removed after it sat idle past retention is NOT_FOUND too, so such a session starts over empty.
    if (latestCursor === undefined) {
      await this.publish({ format: FORMAT, type: 'header' });
      return;
    }

    // subscribe() never ends on its own, so the read stops at the newest message the metadata reported.
    const abort = new AbortController();
    let sawHeader = false;
    let previousSeq = 0;
    let caughtUp = false;
    try {
      for await (const event of streams.subscribe(this.topic, {
        namespace: this.namespace,
        signal: abort.signal,
      })) {
        const message = decode(event.payload);
        if (!sawHeader) {
          // Retention drops the oldest messages first, so a missing header means commits are missing too.
          if (message.type !== 'header') throw this.expired();
          sawHeader = true;
        } else if (message.type === 'commit') {
          for (const { seq, writes } of message.commits) {
            if (seq <= previousSeq) {
              throw new HatchetStorageError(
                `Commit ${seq} after commit ${previousSeq} in ${this.topic}: two owners wrote this storage`
              );
            }
            try {
              this.memory.prepareCommit(writes, seq as Seq).apply();
            } catch (error) {
              throw new HatchetStorageError(`Invalid commit ${seq} in ${this.topic}`, {
                cause: error,
              });
            }
            previousSeq = seq;
          }
        } else {
          throw new HatchetStorageError(`A second header in ${this.topic}`);
        }
        if (event.cursor === latestCursor) {
          caughtUp = true;
          break;
        }
      }
    } finally {
      abort.abort();
    }
    if (!caughtUp) {
      throw new HatchetStorageError(
        `The read of ${this.topic} ended before its newest message; open it again`
      );
    }
  }

  /**
   * Applies `batches` to memory one by one, each checked against the state the earlier ones left, then publishes
   * them as one message, so they are stored together or not at all. A failed publish may still have stored them,
   * and memory is ahead of the topic either way, so it poisons the storage: pi then fails the session, and the
   * next owner reads whatever was stored.
   */
  async commit(
    batches: readonly (readonly StorageWrite[])[],
    receive: (index: number, seq: Seq) => void,
    _context: Context
  ): Promise<CommitResult> {
    this.assertUsable();
    const seqs: Seq[] = [];
    const commits: { seq: number; writes: StorageWrite[] }[] = [];
    let refused: { error: unknown } | undefined;
    let settle = () => {};
    this.writing = new Promise<void>((resolve) => (settle = resolve));
    try {
      for (const writes of batches) {
        let prepared;
        try {
          prepared = this.memory.prepareCommit(writes);
        } catch (error) {
          refused = { error };
          break;
        }
        commits.push({ seq: prepared.seq, writes: [...prepared.writes] });
        seqs.push(prepared.apply());
      }
      if (commits.length > 0) await this.publish({ format: FORMAT, type: 'commit', commits });
    } catch (error) {
      if (seqs.length === 0) throw error;
      throw this.poison(error);
    } finally {
      this.writing = undefined;
      settle();
    }
    for (const [index, seq] of seqs.entries()) receive(index, seq);
    return refused === undefined
      ? { stored: seqs.length }
      : { stored: seqs.length, refused: refused.error };
  }

  async mintId<I extends Id<string>>(): Promise<I> {
    return this.store.mintId<I>();
  }

  async conversation(id: ConversationId, context: Context) {
    return (await this.readable()).conversation(id, context);
  }

  async scanConversations(
    query: ConversationQuery,
    limit: number,
    cursor: Cursor | undefined,
    context: Context
  ) {
    return (await this.readable()).scanConversations(query, limit, cursor, context);
  }

  entry(id: EntryId, context: Context): ReturnType<Storage['entry']>;
  entry(
    conversationId: ConversationId,
    id: EntryId,
    context: Context
  ): ReturnType<Storage['entry']>;
  async entry(
    first: EntryId | ConversationId,
    second: EntryId | Context,
    third?: Context
  ): ReturnType<Storage['entry']> {
    const store = await this.readable();
    return third === undefined
      ? store.entry(first as EntryId, second as Context)
      : store.entry(first as ConversationId, second as EntryId, third);
  }

  async findLatestHeadMarker(
    conversationId: ConversationId,
    atOrBeforeEntryId: EntryId | undefined,
    context: Context
  ) {
    return (await this.readable()).findLatestHeadMarker(conversationId, atOrBeforeEntryId, context);
  }

  async scanEntries(
    query: EntryQuery,
    limit: number,
    cursor: Cursor | undefined,
    context: Context
  ) {
    return (await this.readable()).scanEntries(query, limit, cursor, context);
  }

  async task(id: TaskId, context: Context): ReturnType<Storage['task']> {
    return (await this.readable()).task(id, context);
  }

  async scanTasks(
    query: TaskQuery,
    limit: number,
    cursor: Cursor | undefined,
    context: Context
  ): ReturnType<Storage['scanTasks']> {
    return (await this.readable()).scanTasks(query, limit, cursor, context);
  }

  async submission(id: SubmissionId, context: Context) {
    return (await this.readable()).submission(id, context);
  }

  async scanSubmissions(
    query: SubmissionQuery,
    limit: number,
    cursor: Cursor | undefined,
    context: Context
  ) {
    return (await this.readable()).scanSubmissions(query, limit, cursor, context);
  }

  async submissionByRequest(conversationId: ConversationId, requestId: string, context: Context) {
    return (await this.readable()).submissionByRequest(conversationId, requestId, context);
  }

  async findDocument(address: DocumentAddress, at: DocumentPoint, context: Context) {
    return (await this.readable()).findDocument(address, at, context);
  }

  async document(id: DocumentId, at: DocumentPoint, context: Context) {
    return (await this.readable()).document(id, at, context);
  }

  async scanDocuments(
    query: DocumentQuery,
    limit: number,
    cursor: Cursor | undefined,
    context: Context
  ) {
    return (await this.readable()).scanDocuments(query, limit, cursor, context);
  }

  async close(context: Context): Promise<void> {
    if (this.closed) return;
    this.closed = true;
    await this.memory.close(context);
  }

  private async publish(message: Message): Promise<void> {
    await this.hatchet.streams.publish(this.topic, JSON.stringify(message), {
      namespace: this.namespace,
    });
  }

  private expired(): HatchetStorageError {
    return new HatchetStorageError(
      `${this.topic} has lost its oldest commits to the tenant's stream retention`
    );
  }

  private get store(): MemoryStorage {
    this.assertUsable();
    return this.memory;
  }

  /** Memory runs ahead of the topic while a commit is being published, so reads wait for it. */
  private async readable(): Promise<MemoryStorage> {
    while (this.writing !== undefined) await this.writing;
    return this.store;
  }

  private poison(cause: unknown): Error {
    this.poisonError ??= new HatchetStorageError(
      `A commit to ${this.topic} may not be stored; reopen the storage`,
      { cause }
    );
    return this.poisonError;
  }

  private assertUsable(): void {
    if (this.closed) throw new HatchetStorageError('The storage is closed');
    if (this.poisonError !== undefined) throw this.poisonError;
  }
}

function decode(payload: Uint8Array): Message {
  const message = JSON.parse(new TextDecoder().decode(payload)) as Message;
  if (message.format !== FORMAT) {
    throw new HatchetStorageError(`Unsupported storage message format ${String(message.format)}`);
  }
  return message;
}

function errorCode(error: unknown): unknown {
  return typeof error === 'object' && error !== null && 'code' in error ? error.code : undefined;
}
