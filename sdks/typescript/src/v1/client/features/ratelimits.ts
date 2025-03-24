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
 * RatelimitsClient is used to manage rate limits for the Hatchet
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

  async upsert(opts: CreateRateLimitOpts): Promise<string> {
    await this.admin.putRateLimit(opts.key, opts.limit, opts.duration);
    return opts.key;
  }

  async list(opts: Parameters<typeof this.api.rateLimitList>[1]) {
    const { data } = await this.api.rateLimitList(this.tenantId, opts);
    return data;
  }

  // FIXME: there's no delete for rate limits
  // FIXME: nice to have refresh to set value to 0
}
