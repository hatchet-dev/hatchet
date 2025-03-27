import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { V1TaskStatus, V1TaskFilter } from '@hatchet/clients/rest/generated/data-contracts';
import { WorkflowsClient } from './workflows';
import { HatchetClient } from '../client';

export type RunFilter = {
  since: Date;
  until?: Date;
  statuses?: V1TaskStatus[];
  workflowNames?: string[];
  additionalMetadata?: Record<string, string>;
};

export type CancelRunOpts = {
  ids?: string[];
  filters?: RunFilter;
};

export type ReplayRunOpts = {
  ids?: string[];
  filters?: RunFilter;
};

/**
 * RunsClient is used to list and manage runs
 */
export class RunsClient {
  api: HatchetClient['api'];
  tenantId: string;
  workflows: WorkflowsClient;

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.tenantId = client.tenantId;
    this.workflows = client.workflows;
  }

  async get<T = any>(run: string | WorkflowRunRef<T>) {
    const runId = typeof run === 'string' ? run : await run.getWorkflowRunId();

    const { data } = await this.api.workflowRunGet(this.tenantId, runId);
    return data;
  }

  async getDetails<T = any>(run: string | WorkflowRunRef<T>) {
    const runId = typeof run === 'string' ? run : await run.getWorkflowRunId();

    const { data } = await this.api.workflowRunGetShape(this.tenantId, runId);
    return data;
  }

  async list(opts?: Parameters<typeof this.api.workflowRunList>[1]) {
    // TODO workflow id on opts is a uuid

    const { data } = await this.api.workflowRunList(this.tenantId, opts);
    return data;
  }

  async cancel(opts: CancelRunOpts) {
    const filter = opts.filters && (await this.prepareFilter(opts.filters));
    return this.api.v1TaskCancel(this.tenantId, {
      externalIds: opts.ids,
      filter,
    });
  }

  async replay(opts: ReplayRunOpts) {
    const filter = opts.filters && (await this.prepareFilter(opts.filters));
    return this.api.v1TaskReplay(this.tenantId, {
      externalIds: opts.ids,
      filter,
    });
  }

  private async prepareFilter({
    since,
    until,
    statuses,
    workflowNames,
    additionalMetadata,
  }: RunFilter): Promise<V1TaskFilter> {
    return {
      since: since.toISOString(),
      until: until?.toISOString(),
      statuses,
      workflowIds: await Promise.all(
        workflowNames?.map(async (name) => (await this.workflows.get(name)).metadata.id) || []
      ),
      additionalMetadata: Object.entries(additionalMetadata || {}).map(
        ([key, value]) => `${key}:${value}`
      ),
    };
  }
}
