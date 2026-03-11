import { V1LogLineLevel, V1LogLineOrderByDirection } from '@hatchet/clients/rest/generated/data-contracts';
import { HatchetClient } from '../client';

/**
 * The options for the list logs operation.
 */
export type ListLogsOpts = {
  /** The maximum number of log lines to return. */
  limit?: number;
  /** Return only logs after this date. */
  since?: Date;
  /** Return only logs before this date. */
  until?: Date;
  /** Filter logs by a search string. */
  search?: string;
  /** Filter logs by log level. */
  levels?: V1LogLineLevel[];
  /** The direction to order the logs by. */
  orderByDirection?: V1LogLineOrderByDirection;
  /** Filter logs by attempt number. */
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
