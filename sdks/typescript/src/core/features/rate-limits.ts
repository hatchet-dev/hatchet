import { retrier, type RetrierConfig } from '@hatchet/util/retrier';
import type { Logger } from '@hatchet/util/logger/logger';
import type { WorkflowServiceClient } from '@hatchet/protoc/workflows/workflows';
import type { CreateRateLimitOpts } from '../types';

export interface RateLimitsClientDeps {
  rpc: WorkflowServiceClient;
  logger: Logger;
  retrier?: RetrierConfig;
}

/** Tenant-wide rate limits that tasks declare against by key. */
export class RateLimitsClient {
  constructor(private readonly deps: RateLimitsClientDeps) {}

  /**
   * Creates or updates a rate limit.
   * @returns the rate limit's key
   */
  async put(opts: CreateRateLimitOpts): Promise<string> {
    await retrier(
      () =>
        this.deps.rpc.putRateLimit({ key: opts.key, limit: opts.limit, duration: opts.duration }),
      this.deps.logger,
      this.deps.retrier
    );
    return opts.key;
  }
}
