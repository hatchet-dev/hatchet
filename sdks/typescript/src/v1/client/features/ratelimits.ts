import { RateLimitDuration } from '@hatchet/protoc/workflows';
import {
  RateLimitOrderByField,
  RateLimitOrderByDirection,
} from '@hatchet/clients/rest/generated/data-contracts';
import { HatchetClient } from '../client';

export { RateLimitDuration, RateLimitOrderByField, RateLimitOrderByDirection };

// TODO generate this and transformer to RateLimitDuration with AI
// type RateLimitDurationString = 'second';
// "second" |
// "minute" |
// "hour"

export type CreateRateLimitOpts = {
  key: string;
  limit: number;
  duration?: RateLimitDuration;
};

/**
 * The rate limits client is a wrapper for Hatchet’s gRPC API that makes it easier to work with rate limits in Hatchet.
 */
export class RatelimitsClient {
  api: HatchetClient['api'];
  admin: HatchetClient['admin'];
  tenantId: string;

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.admin = client.admin;
    this.tenantId = client.tenantId;
  }

  /**
   * Upserts a rate limit for the current tenant.
   * @param opts - The options for the upsert operation.
   * @returns A promise that resolves to the key of the upserted rate limit.
   */
  async upsert(opts: CreateRateLimitOpts): Promise<string> {
    await this.admin.putRateLimit(opts.key, opts.limit, opts.duration);
    return opts.key;
  }

  /**
   * Lists all rate limits for the current tenant.
   * @param opts - The options for the list operation.
   * @returns A promise that resolves to the list of rate limits.
   */
  async list(opts: Parameters<typeof this.api.rateLimitList>[1]) {
    const { data } = await this.api.rateLimitList(this.tenantId, opts);
    return data;
  }

  // FIXME: there's no delete for rate limits
  // FIXME: nice to have refresh to set value to 0
}
