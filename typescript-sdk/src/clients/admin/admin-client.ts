import { Channel, ClientFactory } from 'nice-grpc';
import {
  CreateWorkflowVersionOpts,
  WorkflowServiceClient,
  WorkflowServiceDefinition,
} from '@hatchet/protoc/workflows';
import HatchetError from '@util/errors/hatchet-error';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import { Api } from '../rest';

export class AdminClient {
  config: ClientConfig;
  client: WorkflowServiceClient;
  api: Api;
  tenantId: string;

  constructor(
    config: ClientConfig,
    channel: Channel,
    factory: ClientFactory,
    api: Api,
    tenantId: string
  ) {
    this.config = config;
    this.client = factory.create(WorkflowServiceDefinition, channel);
    this.api = api;
    this.tenantId = tenantId;
  }

  async put_workflow(workflow: CreateWorkflowVersionOpts) {
    try {
      await this.client.putWorkflow({
        opts: workflow,
      });
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  async run_workflow(workflowName: string, input: object) {
    try {
      const inputStr = JSON.stringify(input);

      const resp = await this.client.triggerWorkflow({
        name: workflowName,
        input: inputStr,
      });

      return resp.workflowRunId;
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  async list_workflows() {
    return this.api.workflowList(this.tenantId);
  }

  async get_workflow(workflowId: string) {
    return this.api.workflowGet(workflowId);
  }

  async get_workflow_version(workflowId: string) {
    return this.api.workflowVersionGet(workflowId);
  }

  async get_workflow_run(workflowRunId: string) {
    return this.api.workflowRunGet(this.tenantId, workflowRunId);
  }

  async list_workflow_runs(query: {
    offset?: number | undefined;
    limit?: number | undefined;
    eventId?: string | undefined;
    workflowId?: string | undefined;
  }) {
    return this.api.workflowRunList(this.tenantId, query);
  }

  schedule_workflow(workflowId: string, options?: { schedules?: Date[] }) {
    try {
      this.client.scheduleWorkflow({
        workflowId,
        schedules: options?.schedules,
      });
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }
}
