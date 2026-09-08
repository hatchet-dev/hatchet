import { REQUEST_MAX_AGE_SECONDS } from './contract';

/**
 * A bounded set of upgrade nonces seen within the request window. Entries expire with the
 * window and the oldest is dropped past the capacity, so memory stays bounded whatever the
 * operator sends. It lives in one isolate: a replay that lands in another isolate is not
 * caught, which is why a production endpoint should consume nonces in a Durable Object or
 * an expiring KV key through the handler's `seenNonce` option.
 */
export class NonceSet {
  private readonly entries = new Map<string, number>();

  constructor(
    private readonly capacity = 4096,
    private readonly ttlSeconds = REQUEST_MAX_AGE_SECONDS
  ) {}

  /**
   * Records the nonce and reports whether it was already present. Expired entries are
   * dropped first, so a nonce older than the window counts as new again; its signature would
   * be refused for the stale timestamp anyway.
   */
  consume(nonce: string, nowSeconds = Math.floor(Date.now() / 1000)): boolean {
    for (const [seen, expiresAt] of this.entries) {
      if (expiresAt > nowSeconds) {
        break;
      }

      this.entries.delete(seen);
    }

    if (this.entries.has(nonce)) {
      return true;
    }

    this.entries.set(nonce, nowSeconds + this.ttlSeconds);

    while (this.entries.size > this.capacity) {
      const oldest = this.entries.keys().next().value;

      if (oldest === undefined) {
        break;
      }

      this.entries.delete(oldest);
    }

    return false;
  }

  get size(): number {
    return this.entries.size;
  }
}
