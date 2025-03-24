import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { HatchetClient } from '../client';

/**
 * RunsClient is used to list and manage runs
 */
export class RunsClient {
  api: HatchetClient['api'];
  tenantId: string;

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.tenantId = client.tenantId;
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

  async replay(opts: Parameters<typeof this.api.v1TaskReplay>[1]) {
    // TODO is v1 check
    // TODO   workflowIds?: string[]; on opts.filters
    const { data } = await this.api.v1TaskReplay(this.tenantId, opts);
    return data;
  }

  async cancel(opts: Parameters<typeof this.api.v1TaskCancel>[1]) {
    // TODO is v1 check
    // TODO   workflowIds?: string[]; on opts.filters
    const { data } = await this.api.v1TaskCancel(this.tenantId, opts);
    return data;
  }
}
