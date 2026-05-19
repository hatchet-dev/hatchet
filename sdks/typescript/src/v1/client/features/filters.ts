import { HatchetClient } from '../client';

export type WorkflowIdScopePair = {
  workflowId: string;
  scope: string;
};

/**
 * The filters client is a client for interacting with Hatchet's filters API.
 */
export class FiltersClient {
  tenantId: string;
  api: HatchetClient['api'];

  constructor(client: HatchetClient) {
    this.tenantId = client.tenantId;
    this.api = client.api;
  }

  /**
   * Lists all filters.
   * @param opts - The options for the list operation.
   * @param opts.limit - The number of filters to return.
   * @param opts.offset - The number of filters to skip before returning the result set.
   * @param opts.workflowIds - A list of workflow IDs to filter by.
   * @param opts.scopes - A list of scopes to filter by.
   * @returns A promise that resolves to the list of filters.
   */
  async list(opts?: {
    limit?: number;
    offset?: number;
    workflowIds?: string[];
    scopes?: string[];
  }) {
    const { data } = await this.api.v1FilterList(this.tenantId, {
      limit: opts?.limit,
      offset: opts?.offset,
      workflowIds: opts?.workflowIds,
      scopes: opts?.scopes,
    });

    return data;
  }

  /**
   * Gets a filter by its ID.
   * @param filterId - The ID of the filter to get.
   * @returns A promise that resolves to the filter.
   */
  async get(filterId: Parameters<typeof this.api.v1FilterGet>[1]) {
    const { data } = await this.api.v1FilterGet(this.tenantId, filterId);
    return data;
  }

  /**
   * Creates a new filter.
   * @param opts - The options for the create operation.
   * @returns A promise that resolves to the created filter.
   */
  async create(opts: Parameters<typeof this.api.v1FilterCreate>[1]) {
    const { data } = await this.api.v1FilterCreate(this.tenantId, opts);
    return data;
  }

  /**
   * Deletes a filter by its ID.
   * @param filterId - The ID of the filter to delete.
   * @returns A promise that resolves to the deleted filter.
   */
  async delete(filterId: Parameters<typeof this.api.v1FilterDelete>[1]) {
    const { data } = await this.api.v1FilterDelete(this.tenantId, filterId);
    return data;
  }

  /**
   * Updates a filter by its ID.
   * @param filterId - The ID of the filter to update.
   * @param updates - The updates to apply to the filter.
   * @returns A promise that resolves to the updated filter.
   */
  async update(
    filterId: Parameters<typeof this.api.v1FilterDelete>[1],
    updates: Parameters<typeof this.api.v1FilterUpdate>[2]
  ) {
    const { data } = await this.api.v1FilterUpdate(this.tenantId, filterId, updates);
    return data;
  }
}
