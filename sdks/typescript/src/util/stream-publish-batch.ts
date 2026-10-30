import HatchetError from '@util/errors/hatchet-error';

/** The most publishes one `PublishBatch` request carries; the engine sets no limit of its own. */
export const MAX_PUBLISH_BATCH_ENTRIES = 500;

/**
 * The most inline payload bytes one `PublishBatch` request carries, counting each entry's
 * framing, so the request stays under gRPC's 4 MiB message limit with the same headroom a
 * single max-size publish has (`MaxStreamMessagePayloadBytes`).
 */
export const MAX_PUBLISH_BATCH_BYTES = 4 * 1024 * 1024 - 5 * 1024;

// an entry's tag and length prefixes, rounded up
const entryOverheadBytes = 16;

// setTimeout fires at once for a longer delay, so a longer wait takes several
const maxTimerMs = 2 ** 31 - 1;

/**
 * Checks a publish's batch timeout before anything is queued or uploaded, so a bad one fails
 * only its own publish.
 */
export function validateBatchTimeout(batchTimeoutMs: number | undefined): void {
  if (batchTimeoutMs !== undefined && !(Number.isFinite(batchTimeoutMs) && batchTimeoutMs >= 0)) {
    throw new HatchetError(
      `batchTimeoutMs must be a finite number of milliseconds, 0 or more; got ${batchTimeoutMs}`
    );
  }
}

/**
 * One topic's publishes waiting to be sent. A request goes out once it is full or once any
 * publish in the queue has waited its batch timeout.
 */
export class StreamPublishQueue<T> {
  private readonly items: Array<{ item: T; bytes: number; sendBy: number }> = [];

  private wake: (() => void) | undefined;

  /** @param payloadBytes - what an item adds to a request */
  constructor(private readonly payloadBytes: (item: T) => number) {}

  get size(): number {
    return this.items.length;
  }

  push(item: T, batchTimeoutMs = 0): void {
    this.items.push({ item, bytes: this.payloadBytes(item), sendBy: Date.now() + batchTimeoutMs });
    // a new publish may fill the request or have a shorter timeout
    this.wake?.();
  }

  /**
   * Resolves with the next request's publishes, at least one. Call it only while the queue
   * holds publishes; they leave it only through this.
   */
  async next(): Promise<T[]> {
    for (;;) {
      const count = this.fitting();
      const full = count < this.items.length || count === MAX_PUBLISH_BATCH_ENTRIES;
      // a loop, since spreading a long queue into Math.min overflows the call stack
      let sendBy = Infinity;
      for (const i of this.items) {
        sendBy = Math.min(sendBy, i.sendBy);
      }

      const wait = sendBy - Date.now();
      if (full || wait <= 0) {
        return this.items.splice(0, count).map((i) => i.item);
      }

      await new Promise<void>((resolve) => {
        const timer = setTimeout(done, Math.min(wait, maxTimerMs));
        function done() {
          clearTimeout(timer);
          resolve();
        }
        this.wake = done;
      });
      this.wake = undefined;
    }
  }

  // always at least one, so a max-size payload still goes out on its own
  private fitting(): number {
    let bytes = 0;
    let count = 0;

    while (count < this.items.length && count < MAX_PUBLISH_BATCH_ENTRIES) {
      const next = this.items[count].bytes + entryOverheadBytes;
      if (count > 0 && bytes + next > MAX_PUBLISH_BATCH_BYTES) {
        break;
      }
      bytes += next;
      count += 1;
    }

    return count;
  }
}
