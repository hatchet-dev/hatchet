import { HatchetClient } from '../client';

export type TaskStatusMetrics = {
  cancelled: number;
  completed: number;
  failed: number;
  queued: number;
  running: number;
};

/**
 * MetricsClient is used to get metrics for workflows
 */
export class MetricsClient {
  tenantId: string;
  api: HatchetClient['api'];

  constructor(client: HatchetClient) {
    this.tenantId = client.tenantId;
    this.api = client.api;
  }

  /**
   * Get task/run status metrics for a tenant.
   *
   * This backs the dashboard "runs list" status count badges.
   *
   * Endpoint: GET /api/v1/stable/tenants/{tenant}/task-metrics
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

  async getQueueMetrics(opts?: Parameters<typeof this.api.tenantGetStepRunQueueMetrics>[1]) {
    const { data } = await this.api.tenantGetStepRunQueueMetrics(this.tenantId, opts);
    return data;
  }
}
