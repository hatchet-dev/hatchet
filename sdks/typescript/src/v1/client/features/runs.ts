import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { V1TaskStatus, V1TaskFilter } from '@hatchet/clients/rest/generated/data-contracts';
import {
  RunEventType,
  RunListenerClient,
} from '@hatchet/clients/listeners/run-listener/child-listener-client';
import { WorkflowsClient } from './workflows';
import { HatchetClient } from '../client';

export type RunFilter = {
  since?: Date;
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

export interface ListRunsOpts extends RunFilter {
  /**
   * The number to skip
   * @format int64
   */
  offset?: number;
  /**
   * The number to limit by
   * @format int64
   */
  limit?: number;
  /** A list of statuses to filter by */

  /**
   * The worker id to filter by
   * @format uuid
   * @minLength 36
   * @maxLength 36
   */
  workerId?: string;
  /** Whether to include DAGs or only to include tasks */
  onlyTasks: boolean;

  /**
   * The parent task run external id to filter by
   * @deprecated use parentTaskRunExternalId instead
   * @format uuid
   * @minLength 36
   * @maxLength 36
   */
  parentTaskExternalId?: string;

  /**
   * The parent task run external id to filter by
   * @format uuid
   * @minLength 36
   * @maxLength 36
   */
  parentTaskRunExternalId?: string;

  /**
   * The triggering event external id to filter by
   * @format uuid
   * @minLength 36
   * @maxLength 36
   */
  triggeringEventExternalId?: string;

  /** A flag for whether or not to include the input and output payloads in the response. Defaults to `true` if unset. */
  includePayloads?: boolean;
}

/**
 * The runs client is a client for interacting with task and workflow runs within Hatchet.
 */
export class RunsClient {
  api: HatchetClient['api'];
  tenantId: string;
  workflows: WorkflowsClient;
  listener: RunListenerClient;

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.tenantId = client.tenantId;
    this.workflows = client.workflows;

    this.listener = client._listener;
  }

  /**
   * Gets a task or workflow run by its ID.
   * @param run - The ID of the run to get.
   * @returns A promise that resolves to the run.
   */
  async get<T = any>(run: string | WorkflowRunRef<T>) {
    const runId = typeof run === 'string' ? run : await run.getWorkflowRunId();

    const { data } = await this.api.v1WorkflowRunGet(runId);
    return data;
  }

  /**
   * Gets the status of a task or workflow run by its ID.
   * @param run - The ID of the run to get the status of.
   * @returns A promise that resolves to the status of the run.
   */
  async get_status<T = any>(run: string | WorkflowRunRef<T>) {
    const runId = typeof run === 'string' ? run : await run.getWorkflowRunId();

    const { data } = await this.api.v1WorkflowRunGetStatus(runId);
    return data;
  }

  /**
   * Lists all task and workflow runs for the current tenant.
   * @param opts - The options for the list operation.
   * @returns A promise that resolves to the list of runs.
   */
  async list(opts?: Partial<ListRunsOpts>) {
    const normalizedOpts =
      opts?.parentTaskExternalId && !opts?.parentTaskRunExternalId
        ? { ...opts, parentTaskRunExternalId: opts.parentTaskExternalId }
        : opts;

    const { data } = await this.api.v1WorkflowRunList(this.tenantId, {
      ...(await this.prepareListFilter(normalizedOpts || {})),
    });
    return data;
  }

  /**
   * Cancels a task or workflow run by its ID.
   * @param opts - The options for the cancel operation.
   * @returns A promise that resolves to the cancelled run.
   */
  async cancel(opts: CancelRunOpts) {
    const filter = await this.prepareFilter(opts.filters || {});

    return this.api.v1TaskCancel(this.tenantId, {
      externalIds: opts.ids,
      filter: !opts.ids ? filter : undefined,
    });
  }

  /**
   * Replays a task or workflow run by its ID.
   * @param opts - The options for the replay operation.
   * @returns A promise that resolves to the replayed run.
   */
  async replay(opts: ReplayRunOpts) {
    const filter = await this.prepareFilter(opts.filters || {});
    return this.api.v1TaskReplay(this.tenantId, {
      externalIds: opts.ids,
      filter: !opts.ids ? filter : undefined,
    });
  }

  private async prepareFilter({
    since,
    until,
    statuses,
    workflowNames,
    additionalMetadata,
  }: Partial<RunFilter>): Promise<V1TaskFilter> {
    const am = Object.entries(additionalMetadata || {}).map(([key, value]) => `${key}:${value}`);

    return {
      // default to 1 hour ago
      since: since ? since.toISOString() : new Date(Date.now() - 1000 * 60 * 60).toISOString(),
      until: until?.toISOString(),
      statuses,
      workflowIds: await Promise.all(
        workflowNames?.map(async (name) => (await this.workflows.get(name)).metadata.id) || []
      ),
      additionalMetadata: am,
    };
  }

  private async prepareListFilter(
    opts: Partial<ListRunsOpts>
  ): Promise<Parameters<typeof this.api.v1WorkflowRunList>[1]> {
    const am = Object.entries(opts.additionalMetadata || {}).map(
      ([key, value]) => `${key}:${value}`
    );

    return {
      offset: opts.offset,
      limit: opts.limit,
      // default to 1 hour ago
      since: opts.since
        ? opts.since.toISOString()
        : new Date(Date.now() - 1000 * 60 * 60).toISOString(),
      until: opts.until?.toISOString(),
      statuses: opts.statuses,
      worker_id: opts.workerId,
      workflow_ids: await Promise.all(
        opts.workflowNames?.map(async (name) => (await this.workflows.get(name)).metadata.id) || []
      ),
      additional_metadata: am,
      only_tasks: opts.onlyTasks || false,
      parent_task_external_id: opts.parentTaskRunExternalId,
      triggering_event_external_id: opts.triggeringEventExternalId,
      include_payloads: opts.includePayloads,
    };
  }

  /**
   * Creates a run reference for a task or workflow run by its ID.
   * @param id - The ID of the run to create a reference for.
   * @returns A promise that resolves to the run reference.
   */
  runRef<T extends Record<string, any> = any>(id: string): WorkflowRunRef<T> {
    return new WorkflowRunRef<T>(id, this.listener, this);
  }

  /**
   * Subscribes to a stream of events for a task or workflow run by its ID.
   * @param workflowRunId - The ID of the run to subscribe to.
   * @returns A promise that resolves to the stream of events.
   */
  async *subscribeToStream(workflowRunId: string): AsyncIterableIterator<string> {
    const ref = this.runRef(workflowRunId);
    const stream = await ref.stream();

    for await (const event of stream) {
      if (event.type === RunEventType.STEP_RUN_EVENT_TYPE_STREAM) {
        yield event.payload;
      }
    }
  }
}
