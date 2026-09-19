import { REQUEST_MAX_AGE_SECONDS } from './contract';

/** What consuming a nonce found. */
export type NonceOutcome = 'accepted' | 'replayed' | 'full';

/**
 * The upgrade nonces accepted within the request window. Every accepted nonce is kept until
 * its window expires, so a captured upgrade cannot be replayed while its signature is still
 * fresh; nothing is dropped early to make room. When the set holds `capacity` unexpired
 * nonces a new one is refused rather than admitted, and the handler answers 503 so the
 * operator retries once entries have expired. The set lives in one isolate: a replay that
 * lands in another isolate is not caught, which is why a production endpoint should
 * consume nonces in a Durable Object or an expiring KV key through the handler's
 * `seenNonce` option.
 */
export class NonceSet {
  private readonly entries = new Map<string, number>();

  constructor(
    private readonly capacity = 4096,
    private readonly ttlSeconds = REQUEST_MAX_AGE_SECONDS
  ) {}

  /**
   * Records the nonce unless it is present already or the set is full. Expired entries are
   * dropped first; a nonce older than the window counts as new again, and its signature
   * would be refused for the stale timestamp anyway.
   */
  consume(nonce: string, nowSeconds = Math.floor(Date.now() / 1000)): NonceOutcome {
    for (const [seen, expiresAt] of this.entries) {
      if (expiresAt > nowSeconds) {
        break;
      }

      this.entries.delete(seen);
    }

    if (this.entries.has(nonce)) {
      return 'replayed';
    }

    if (this.entries.size >= this.capacity) {
      return 'full';
    }

    this.entries.set(nonce, nowSeconds + this.ttlSeconds);

    return 'accepted';
  }

  get size(): number {
    return this.entries.size;
  }
}
