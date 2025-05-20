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

  async list(opts?: {
    limit?: number;
    offset?: number;
    workflowIdsAndScopes?: WorkflowIdScopePair[];
  }) {
    const hasWorkflowIdsAndScopes = opts?.workflowIdsAndScopes !== undefined;
    const workflowIds = hasWorkflowIdsAndScopes
      ? opts.workflowIdsAndScopes?.map((pair) => pair.workflowId)
      : undefined;
    const scopes = hasWorkflowIdsAndScopes
      ? opts.workflowIdsAndScopes?.map((pair) => pair.scope)
      : undefined;

    const { data } = await this.api.v1FilterList(this.tenantId, {
      limit: opts?.limit,
      offset: opts?.offset,
      workflowIds,
      scopes,
    });

    return data;
  }

  async get(filterId: Parameters<typeof this.api.v1FilterGet>[1]) {
    const { data } = await this.api.v1FilterGet(this.tenantId, filterId);
    return data;
  }

  async create(opts: Parameters<typeof this.api.v1FilterCreate>[1]) {
    const { data } = await this.api.v1FilterCreate(this.tenantId, opts);
    return data;
  }

  async delete(filterId: Parameters<typeof this.api.v1FilterDelete>[1]) {
    const { data } = await this.api.v1FilterDelete(this.tenantId, filterId);
    return data;
  }
}
