/** What consuming a nonce found. */
export type NonceOutcome = 'accepted' | 'replayed' | 'full';

/**
 * The upgrade nonces accepted while their signatures are fresh. Every accepted nonce is kept
 * through the last second its signature is accepted (the signed timestamp plus the request
 * window, which the verifier passes in), so a captured upgrade cannot be replayed while its
 * signature still verifies, including one signed ahead of this clock; nothing is dropped
 * early to make room. When the set holds `capacity` unexpired nonces a new one is refused
 * rather than admitted, and the handler answers 503 so the operator retries once entries
 * have expired. The set lives in one isolate: a replay that lands in another isolate is not
 * caught, which is why a production endpoint should consume nonces in a Durable Object or
 * an expiring KV key through the handler's `seenNonce` option.
 */
export class NonceSet {
  private readonly entries = new Map<string, number>();

  constructor(private readonly capacity = 4096) {}

  /**
   * Records the nonce unless it is present already or the set is full. Expired entries are
   * dropped first; `expiresAtUnixSeconds` is inclusive, so a nonce is forgotten once the
   * clock is past it, when its signature is stale anyway.
   */
  consume(
    nonce: string,
    expiresAtUnixSeconds: number,
    nowSeconds = Math.floor(Date.now() / 1000)
  ): NonceOutcome {
    for (const [seen, expiresAt] of this.entries) {
      if (expiresAt < nowSeconds) {
        this.entries.delete(seen);
      }
    }

    if (this.entries.has(nonce)) {
      return 'replayed';
    }

    if (this.entries.size >= this.capacity) {
      return 'full';
    }

    this.entries.set(nonce, expiresAtUnixSeconds);

    return 'accepted';
  }

  get size(): number {
    return this.entries.size;
  }
}
