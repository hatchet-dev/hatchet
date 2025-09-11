import { BaseWorkflowDeclaration, workflowNameString } from '@hatchet/v1';
import { Workflow } from '@hatchet/workflow';
import { HatchetClient } from '../client';
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

  async getWorkflowMetrics(
    workflow: string | Workflow | BaseWorkflowDeclaration<any, any>,
    opts?: Parameters<typeof this.api.workflowGetMetrics>[1]
  ) {
    const name = workflowNameString(workflow);

    const { data } = await this.api.workflowGetMetrics(name, opts);
    return data;
  }

  async getQueueMetrics(
    opts?: Parameters<typeof this.api.tenantGetQueueMetrics>[1] & {
      // TODO override the workflows to this...
      // workflows?: (string | WorkflowDeclaration<any, any> | Workflow)[];
    }
  ) {
    // TODO IMPORTANT workflow id is the uuid for the workflow... not its name
    // const stringWorkflows = opts?.workflows?

    const { data } = await this.api.tenantGetQueueMetrics(this.tenantId, {
      ...opts,
      // workflows: stringWorkflows,
    });
    return data;
  }

  async getTaskMetrics(opts?: Parameters<typeof this.api.tenantGetStepRunQueueMetrics>[1]) {
    // TODO what is this...
    const { data } = await this.api.tenantGetStepRunQueueMetrics(this.tenantId, opts);
    return data;
  }
}
