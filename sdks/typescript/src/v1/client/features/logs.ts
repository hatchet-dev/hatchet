import { V1LogLineLevel, V1LogLineOrderByDirection } from '@hatchet/clients/rest/generated/data-contracts';
import { HatchetClient } from '../client';


export type ListLogsOpts = {
  limit?: number;
  since?: Date;
  until?: Date;
  search?: string;
  levels?: V1LogLineLevel[];
  orderByDirection?: V1LogLineOrderByDirection;
  attempt?: number;
};

/**
 * The logs client is a client for interacting with Hatchet's logs API.
 */
export class LogsClient {
  tenantId: string;
  api: HatchetClient['api'];

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.tenantId = client.tenantId;
  }

  /**
   * Lists the logs for a given task run.
   * @param taskRunId - The ID of the task run to list logs for.
   * @param opts - The options filter for the list operation.
   * @returns A promise that resolves to the list of logs.
   */
  async list(
    taskRunId: string,
    opts?: ListLogsOpts
  ) {
    const { data } = await this.api.v1LogLineList(taskRunId, {
      limit: opts?.limit,
      since: opts?.since?.toISOString(),
      until: opts?.until?.toISOString(),
      search: opts?.search,
      levels: opts?.levels,
      order_by_direction: opts?.orderByDirection,
      attempt: opts?.attempt,
    });
    return data;
  }
}

export { V1LogLineLevel, V1LogLineOrderByDirection };
