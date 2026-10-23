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

export interface QueuedStreamPublish {
  /** What the entry adds to the request. */
  bytes: number;
  /** When, in epoch ms, the publish stops waiting for others to share its request. */
  sendBy: number;
}

/**
 * One topic's publishes waiting to be sent. A request goes out once it is full or once any
 * publish in the queue has waited its batch timeout.
 */
export class StreamPublishQueue<T extends QueuedStreamPublish> {
  private readonly items: T[] = [];

  private wake: (() => void) | undefined;

  get size(): number {
    return this.items.length;
  }

  push(item: T): void {
    this.items.push(item);
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

      let sendBy = Infinity;
      for (const item of this.items) {
        sendBy = Math.min(sendBy, item.sendBy);
      }

      const wait = sendBy - Date.now();
      if (full || wait <= 0) {
        return this.items.splice(0, count);
      }

      await new Promise<void>((resolve) => {
        const timer = setTimeout(done, wait);
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
