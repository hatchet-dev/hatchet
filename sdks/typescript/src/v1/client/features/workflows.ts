import { Workflow } from '@hatchet/workflow';
import { WorkflowDeclaration } from '@hatchet/v1';
import { HatchetClient } from '../client';

export const workflowNameString = (workflow: string | Workflow | WorkflowDeclaration<any, any>) =>
  typeof workflow === 'string' ? workflow : workflow.id;

/**
 * WorkflowsClient is used to list and manage workflows
 */
export class WorkflowsClient {
  api: HatchetClient['api'];
  tenantId: string;

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.tenantId = client.tenantId;
  }

  async get(workflow: string | WorkflowDeclaration<any, any> | Workflow) {
    // TODO name or uuid
    const name = workflowNameString(workflow);

    const { data } = await this.api.workflowGet(name);
    return data;
  }

  async list(opts?: Parameters<typeof this.api.workflowList>[1]) {
    // TODO name or uuid
    const { data } = await this.api.workflowList(this.tenantId, opts);
    return data;
  }

  async delete(workflow: string | WorkflowDeclaration<any, any> | Workflow) {
    // TODO name or uuid
    const name = workflowNameString(workflow);

    const { data } = await this.api.workflowDelete(name);
    return data;
  }

  async isPaused(workflow: string | WorkflowDeclaration<any, any> | Workflow) {
    const wf = await this.get(workflow);
    return wf.isPaused;
  }

  async pause(workflow: string | WorkflowDeclaration<any, any> | Workflow) {
    // TODO name or uuid
    const name = workflowNameString(workflow);

    const { data } = await this.api.workflowUpdate(name, {
      isPaused: true,
    });
    return data;
  }

  async unpause(workflow: string | WorkflowDeclaration<any, any> | Workflow) {
    // TODO name or uuid
    const name = workflowNameString(workflow);

    const { data } = await this.api.workflowUpdate(name, {
      isPaused: false,
    });
    return data;
  }
}
