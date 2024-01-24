import { ServerError, Status, createChannel, createClient } from 'nice-grpc';
import {
  CreateWorkflowVersionOpts,
  Workflow,
  WorkflowServiceClient,
  WorkflowServiceDefinition,
} from '@protoc/workflows';
import HatchetError from '@util/errors/hatchet-error';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import * as semver from '@util/semver/semver';

export class AdminClient {
  config: ClientConfig;
  client: WorkflowServiceClient;

  constructor(config: ClientConfig) {
    this.config = config;
    this.client = createClient(
      WorkflowServiceDefinition,
      createChannel(config.host_port, config.credentials)
    );
  }

  static should_put(
    workflow: CreateWorkflowVersionOpts,
    existing?: Workflow,
    options?: { autoVersion?: boolean }
  ): [boolean, string] {
    // TODO verify this function before merge #122
    if (!options?.autoVersion) {
      return [false, workflow.version];
    }

    if (workflow.version === '') {
      return [true, 'v0.1.0'];
    }

    if (existing && existing.versions.length > 0) {
      const newVersion = semver.bumpMinorVersion(existing.versions[0].version);
      const shouldPut = newVersion !== workflow.version;
      return [shouldPut, newVersion];
    }

    return [true, workflow.version];
  }

  async put_workflow(workflow: CreateWorkflowVersionOpts, options?: { autoVersion?: boolean }) {
    if (workflow.version === '' && !options?.autoVersion) {
      throw new HatchetError(
        'PutWorkflow error: workflow version is required, or use with_auto_version'
      );
    }

    let existing: Workflow | undefined;

    try {
      existing = await this.client.getWorkflowByName({
        tenantId: this.config.tenant_id,
        name: workflow.name,
      });
    } catch (e: any) {
      if (e instanceof ServerError && e.code === Status.NOT_FOUND) {
        existing = undefined;
      } else {
        throw new HatchetError(e.message);
      }
    }

    const [shouldPut, version] = AdminClient.should_put(workflow, existing, options);

    // eslint-disable-next-line no-param-reassign
    workflow.version = version;

    if (!shouldPut) return;

    try {
      await this.client.putWorkflow({
        tenantId: this.config.tenant_id,
        opts: workflow,
      });
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  schedule_workflow(workflowId: string, options?: { schedules?: Date[] }) {
    try {
      this.client.scheduleWorkflow({
        tenantId: this.config.tenant_id,
        workflowId,
        schedules: options?.schedules,
      });
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }
}
