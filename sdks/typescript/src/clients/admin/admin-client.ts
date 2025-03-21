import { Channel, ClientFactory } from 'nice-grpc';
import {
  BulkTriggerWorkflowRequest,
  CreateWorkflowVersionOpts,
  RateLimitDuration,
  WorkflowServiceClient,
  WorkflowServiceDefinition,
} from '@hatchet/protoc/workflows';
import HatchetError from '@util/errors/hatchet-error';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import { Logger } from '@hatchet/util/logger';
import { retrier } from '@hatchet/util/retrier';
import WorkflowRunRef from '@hatchet/util/workflow-run-ref';

import {
  AdminServiceClient,
  AdminServiceDefinition,
  CreateWorkflowVersionRequest,
} from '@hatchet/protoc/v1/workflows';
import { Api } from '../rest';
import {
  WebhookWorkerCreateRequest,
  WorkflowRunStatus,
  WorkflowRunStatusList,
} from '../rest/generated/data-contracts';
import { ListenerClient } from '../listener/listener-client';

type WorkflowMetricsQuery = {
  workflowId?: string;
  workflowName?: string;
  status?: WorkflowRunStatus;
  groupKey?: string;
};

export type WorkflowRun<T = object> = {
  workflowName: string;
  input: T;
  options?: {
    parentId?: string | undefined;
    parentStepRunId?: string | undefined;
    childIndex?: number | undefined;
    childKey?: string | undefined;
    additionalMetadata?: Record<string, string> | undefined;
  };
};

export class AdminClient {
  config: ClientConfig;
  client: WorkflowServiceClient;
  v1Client: AdminServiceClient;
  api: Api;
  tenantId: string;
  logger: Logger;
  listenerClient: ListenerClient;

  constructor(
    config: ClientConfig,
    channel: Channel,
    factory: ClientFactory,
    api: Api,
    tenantId: string,
    listenerClient: ListenerClient
  ) {
    this.config = config;
    this.client = factory.create(WorkflowServiceDefinition, channel);
    this.v1Client = factory.create(AdminServiceDefinition, channel);
    this.api = api;
    this.tenantId = tenantId;
    this.logger = config.logger(`Admin`, config.log_level);
    this.listenerClient = listenerClient;
  }

  /**
   * @deprecated use putWorkflow instead
   */
  async put_workflow(opts: CreateWorkflowVersionOpts) {
    return this.putWorkflow(opts);
  }

  /**
   * Creates a new workflow or updates an existing workflow. If the workflow already exists, Hatchet will automatically
   * determine if the workflow definition has changed and create a new version if necessary.
   * @param workflow a workflow definition to create
   */
  async putWorkflow(workflow: CreateWorkflowVersionOpts) {
    try {
      return await retrier(async () => this.client.putWorkflow({ opts: workflow }), this.logger);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  /**
   * Creates a new workflow or updates an existing workflow. If the workflow already exists, Hatchet will automatically
   * determine if the workflow definition has changed and create a new version if necessary.
   * @param workflow a workflow definition to create
   */
  async putWorkflowV1(workflow: CreateWorkflowVersionRequest) {
    try {
      return await retrier(async () => this.v1Client.putWorkflow(workflow), this.logger);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  /**
   * @deprecated use putRateLimit instead
   */
  async put_rate_limit(key: string, limit: number, duration: RateLimitDuration) {
    return this.putRateLimit(key, limit, duration);
  }

  async putRateLimit(
    key: string,
    limit: number,
    duration: RateLimitDuration = RateLimitDuration.SECOND
  ) {
    try {
      await retrier(
        async () =>
          this.client.putRateLimit({
            key,
            limit,
            duration,
          }),
        this.logger
      );
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  async registerWebhook(data: WebhookWorkerCreateRequest) {
    return this.api.webhookCreate(this.tenantId, data);
  }

  /**
   * @deprecated use runWorkflow instead
   */
  async run_workflow<T = object>(
    workflowName: string,
    input: T,
    options?: {
      parentId?: string | undefined;
      parentStepRunId?: string | undefined;
      childIndex?: number | undefined;
      childKey?: string | undefined;
      additionalMetadata?: Record<string, string> | undefined;
    }
  ) {
    return this.runWorkflow(workflowName, input, options);
  }

  /**
   * Run a new instance of a workflow with the given input. This will create a new workflow run and return the ID of the
   * new run.
   * @param workflowName the name of the workflow to run
   * @param input an object containing the input to the workflow
   * @param options an object containing the options to run the workflow
   * @returns the ID of the new workflow run
   */
  runWorkflow<Q = object, P = object>(
    workflowName: string,
    input: Q,
    options?: {
      parentId?: string | undefined;
      parentStepRunId?: string | undefined;
      childIndex?: number | undefined;
      childKey?: string | undefined;
      additionalMetadata?: Record<string, string> | undefined;
      desiredWorkerId?: string | undefined;
    }
  ) {
    let computedName = workflowName;

    try {
      if (this.config.namespace && !workflowName.startsWith(this.config.namespace)) {
        computedName = this.config.namespace + workflowName;
      }

      const inputStr = JSON.stringify(input);

      const resp = this.client.triggerWorkflow({
        name: computedName,
        input: inputStr,
        ...options,
        additionalMetadata: options?.additionalMetadata
          ? JSON.stringify(options?.additionalMetadata)
          : undefined,
      });

      return new WorkflowRunRef<P>(resp, this.listenerClient, options?.parentId);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }
  /**
   * Run multiple workflows runs with the given input and options. This will create new workflow runs and return their IDs.
   * Order is preserved in the response.
   * @param workflowRuns an array of objects containing the workflow name, input, and options for each workflow run
   * @returns an array of workflow run references
   */
  runWorkflows<Q = object, P = object>(
    workflowRuns: Array<{
      workflowName: string;
      input: Q;
      options?: {
        parentId?: string | undefined;
        parentStepRunId?: string | undefined;
        childIndex?: number | undefined;
        childKey?: string | undefined;
        additionalMetadata?: Record<string, string> | undefined;
        desiredWorkerId?: string | undefined;
      };
    }>
  ): Promise<WorkflowRunRef<P>[]> {
    // Prepare workflows to be triggered in bulk
    const workflowRequests = workflowRuns.map(({ workflowName, input, options }) => {
      let computedName = workflowName;

      if (this.config.namespace && !workflowName.startsWith(this.config.namespace)) {
        computedName = this.config.namespace + workflowName;
      }

      const inputStr = JSON.stringify(input);

      return {
        name: computedName,
        input: inputStr,
        ...options,
        additionalMetadata: options?.additionalMetadata
          ? JSON.stringify(options.additionalMetadata)
          : undefined,
      };
    });

    try {
      // Call the bulk trigger workflow method
      const bulkTriggerWorkflowResponse = this.client.bulkTriggerWorkflow(
        BulkTriggerWorkflowRequest.create({
          workflows: workflowRequests,
        })
      );

      return bulkTriggerWorkflowResponse.then((res) => {
        return res.workflowRunIds.map((resp, index) => {
          const { options } = workflowRuns[index];
          return new WorkflowRunRef<P>(resp, this.listenerClient, options?.parentId);
        });
      });
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  /**
   * @deprecated use listWorkflows instead
   */
  async list_workflows() {
    return this.listWorkflows();
  }

  /**
   * List workflows in the tenant associated with the API token.
   * @returns a list of all workflows in the tenant
   */
  async listWorkflows() {
    const res = await this.api.workflowList(this.tenantId);
    return res.data;
  }

  /**
   * @deprecated use getWorkflow instead
   */
  async get_workflow(workflowId: string) {
    return this.getWorkflow(workflowId);
  }

  /**
   * Get a workflow by its ID.
   * @param workflowId the workflow ID (**note:** this is not the same as the workflow version id)
   * @returns
   */
  async getWorkflow(workflowId: string) {
    const res = await this.api.workflowGet(workflowId);
    return res.data;
  }

  /**
   * @deprecated use getWorkflowVersion instead
   */
  async get_workflow_version(workflowId: string, version?: string) {
    return this.getWorkflowVersion(workflowId, version);
  }

  /**
   * Get a workflow version.
   * @param workflowId the workflow ID
   * @param version the version of the workflow to get. If not provided, the latest version will be returned.
   * @returns the workflow version
   */
  async getWorkflowVersion(workflowId: string, version?: string) {
    const res = await this.api.workflowVersionGet(workflowId, {
      version,
    });

    return res.data;
  }

  /**
   * @deprecated use getWorkflowRun instead
   */
  async get_workflow_run(workflowRunId: string) {
    return this.getWorkflowRun(workflowRunId);
  }

  /**
   * Get a workflow run.
   * @param workflowRunId the id of the workflow run to get
   * @returns the workflow run
   */
  async getWorkflowRun(workflowRunId: string) {
    return new WorkflowRunRef(workflowRunId, this.listenerClient);
  }

  /**
   * @deprecated use listWorkflowRuns instead
   */
  async list_workflow_runs(query: {
    offset?: number | undefined;
    limit?: number | undefined;
    eventId?: string | undefined;
    workflowId?: string | undefined;
    parentWorkflowRunId?: string | undefined;
    parentStepRunId?: string | undefined;
    statuses?: WorkflowRunStatusList | undefined;
    additionalMetadata?: string[] | undefined;
  }) {
    return this.listWorkflowRuns(query);
  }

  /**
   * List workflow runs in the tenant associated with the API token.
   * @param query the query to filter the list of workflow runs
   * @returns
   */
  async listWorkflowRuns(query: {
    offset?: number | undefined;
    limit?: number | undefined;
    eventId?: string | undefined;
    workflowId?: string | undefined;
    parentWorkflowRunId?: string | undefined;
    parentStepRunId?: string | undefined;
    statuses?: WorkflowRunStatusList | undefined;
    additionalMetadata?: string[] | undefined;
  }) {
    const res = await this.api.workflowRunList(this.tenantId, query);
    return res.data;
  }

  /**
   * @deprecated use scheduleWorkflow instead
   */
  async schedule_workflow(name: string, options?: { schedules?: Date[]; input?: object }) {
    return this.scheduleWorkflow(name, options);
  }

  /**
   * Schedule a workflow to run at a specific time or times.
   * @param name the name of the workflow to schedule
   * @param options an object containing the schedules to set
   * @param input an object containing the input to the workflow
   */
  scheduleWorkflow(name: string, options?: { schedules?: Date[]; input?: object }) {
    let computedName = name;

    try {
      if (this.config.namespace && !name.startsWith(this.config.namespace)) {
        computedName = this.config.namespace + name;
      }

      let input: string | undefined;

      if (options?.input) {
        input = JSON.stringify(options.input);
      }

      this.client.scheduleWorkflow({
        name: computedName,
        schedules: options?.schedules,
        input,
      });
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  /**
   * @deprecated use getWorkflowMetrics instead
   */
  async get_workflow_metrics(data: WorkflowMetricsQuery) {
    return this.getWorkflowMetrics(data);
  }

  /**
   * Get the metrics for a workflow.
   *
   * @param workflowId the ID of the workflow to get metrics for
   * @param workflowName the name of the workflow to get metrics for
   * @param query an object containing query parameters to filter the metrics
   */
  getWorkflowMetrics({ workflowId, workflowName, status, groupKey }: WorkflowMetricsQuery) {
    const params = {
      status,
      groupKey,
    };

    if (workflowName) {
      this.listWorkflows().then((res) => {
        const workflow = res.rows?.find((row) => row.name === workflowName);

        if (workflow) {
          return this.api.workflowGetMetrics(workflow.metadata.id, params);
        }

        throw new Error(`Workflow ${workflowName} not found`);
      });
    } else if (workflowId) {
      return this.api.workflowGetMetrics(workflowId, params);
    }

    throw new Error('Must provide either a workflowId or workflowName');
  }
}
