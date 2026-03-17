import { TaskStats } from '@hatchet/clients/rest/generated/data-contracts';
import { HatchetClient } from '../client';

export type TaskStatusMetrics = {
  cancelled: number;
  completed: number;
  failed: number;
  queued: number;
  running: number;
};

/**
 * The metrics client is a client for reading metrics out of Hatchet’s metrics API.
 */
export class MetricsClient {
  tenantId: string;
  api: HatchetClient['api'];

  constructor(client: HatchetClient) {
    this.tenantId = client.tenantId;
    this.api = client.api;
  }

  /**
   * Returns aggregate task run counts grouped by status (queued, running, completed, failed, cancelled)
   * @param query - Filters for the metrics query (e.g. `since`, `until`, `workflow_ids`).
   * @param requestParams - Optional request-level overrides (headers, signal, etc.).
   * @returns Counts per status for the matched task runs.
   */
  async getTaskStatusMetrics(
    query: Parameters<typeof this.api.v1TaskListStatusMetrics>[1],
    requestParams?: Parameters<typeof this.api.v1TaskListStatusMetrics>[2]
  ): Promise<TaskStatusMetrics> {
    const { data } = await this.api.v1TaskListStatusMetrics(this.tenantId, query, requestParams);
    return data.reduce(
      (acc, curr) => {
        acc[curr.status.toLowerCase() as keyof TaskStatusMetrics] = curr.count;
        return acc;
      },
      {} as Record<keyof TaskStatusMetrics, number>
    );
  }

  /**
   * Returns the queue metrics for the current tenant.
   * @param opts - The options for the request.
   * @returns The queue metrics for the current tenant.
   */
  async getQueueMetrics(opts?: Parameters<typeof this.api.tenantGetStepRunQueueMetrics>[1]) {
    const { data } = await this.api.tenantGetStepRunQueueMetrics(this.tenantId, opts);
    return data;
  }

  /**
   * Scrape Prometheus metrics for the tenant.
   * @returns The metrics in Prometheus text format.
   */
  async scrapePrometheusMetrics(): Promise<string> {
    const { data } = await this.api.tenantGetPrometheusMetrics(this.tenantId);
    return data;
  }

  /**
   * Get task statistics for the tenant.
   * @returns A record mapping task names to their statistics.
   */
  async getTaskStats(): Promise<TaskStats> {
    const { data } = await this.api.tenantGetTaskStats(this.tenantId);
    return data;
  }
}
