import { Channel, ClientFactory } from 'nice-grpc';
import {
  CreateWorkflowVersionOpts,
  WorkflowServiceClient,
  WorkflowServiceDefinition,
} from '@hatchet/protoc/workflows';
import HatchetError from '@util/errors/hatchet-error';
import { ClientConfig } from '@clients/hatchet-client/client-config';

export class AdminClient {
  config: ClientConfig;
  client: WorkflowServiceClient;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory) {
    this.config = config;
    this.client = factory.create(WorkflowServiceDefinition, channel);
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
