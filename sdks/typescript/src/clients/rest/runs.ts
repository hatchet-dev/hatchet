import type { V1TaskFilter } from './generated/data-contracts';
import type { RestApi } from './rest-api';

/** The run filter the REST features take: workflows by name, resolved to ids per call. */
export type RunFilter = {
  /** Defaults to one hour ago. */
  since?: Date;
  until?: Date;
  statuses?: V1TaskFilter['statuses'];
  workflowNames?: string[];
  additionalMetadata?: Record<string, string>;
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

/** Resolves a workflow name to the workflow, the way `WorkflowsClient.get` does. */
export interface WorkflowLookup {
  get(name: string): Promise<{ metadata: { id: string } }>;
}

export interface RestRunsClientDeps {
  api: RestApi;
  tenantId: string;
  workflows: WorkflowLookup;
}

/**
 * The run operations that go over REST, shared by the Node and core run clients: listing and
 * reading runs, and restoring and branching durable tasks.
 */
export class RestRunsClient {
  constructor(private readonly deps: RestRunsClientDeps) {}

  /** A run with its tasks, as the REST API reports it. */
  async get(runId: string) {
    const { data } = await this.deps.api.v1WorkflowRunGet(runId);
    return data;
  }

  /** The status of a run. */
  async getStatus(runId: string) {
    const { data } = await this.deps.api.v1WorkflowRunGetStatus(runId);
    return data;
  }

  /** The runs matching the options; workflow names are resolved to ids first. */
  async list(opts?: Partial<ListRunsOpts>) {
    const normalizedOpts =
      opts?.parentTaskExternalId && !opts?.parentTaskRunExternalId
        ? { ...opts, parentTaskRunExternalId: opts.parentTaskExternalId }
        : opts;

    const { data } = await this.deps.api.v1WorkflowRunList(
      this.deps.tenantId,
      await this.prepareListFilter(normalizedOpts || {})
    );
    return data;
  }

  /** The `V1TaskFilter` of a cancel or replay request; `since` defaults to one hour ago. */
  async prepareFilter({
    since,
    until,
    statuses,
    workflowNames,
    additionalMetadata,
  }: Partial<RunFilter>): Promise<V1TaskFilter> {
    return {
      since: since ? since.toISOString() : new Date(Date.now() - 1000 * 60 * 60).toISOString(),
      until: until?.toISOString(),
      statuses,
      workflowIds: await this.workflowIds(workflowNames),
      additionalMetadata: metadataPairs(additionalMetadata),
    };
  }

  private async prepareListFilter(
    opts: Partial<ListRunsOpts>
  ): Promise<Parameters<RestApi['v1WorkflowRunList']>[1]> {
    return {
      offset: opts.offset,
      limit: opts.limit,
      since: opts.since
        ? opts.since.toISOString()
        : new Date(Date.now() - 1000 * 60 * 60).toISOString(),
      until: opts.until?.toISOString(),
      statuses: opts.statuses,
      worker_id: opts.workerId,
      workflow_ids: await this.workflowIds(opts.workflowNames),
      additional_metadata: metadataPairs(opts.additionalMetadata),
      only_tasks: opts.onlyTasks || false,
      parent_task_external_id: opts.parentTaskRunExternalId,
      triggering_event_external_id: opts.triggeringEventExternalId,
      include_payloads: opts.includePayloads,
    };
  }

  private workflowIds(names: string[] | undefined): Promise<string[]> {
    return Promise.all(
      names?.map(async (name) => (await this.deps.workflows.get(name)).metadata.id) || []
    );
  }

  /**
   * Restores an evicted durable task so it can resume execution.
   * @param taskExternalId - The external ID of the evicted task.
   */
  async restoreTask(taskExternalId: string) {
    return this.deps.api.v1TaskRestore(taskExternalId);
  }

  /**
   * Forks (resets) a durable task from a specific node, triggering re-execution from that point.
   * @param taskExternalId - The external ID of the durable task to reset.
   * @param nodeId - The node ID to replay from.
   */
  async branchDurableTask(taskExternalId: string, nodeId: number, branchId: number = 0) {
    return this.deps.api.v1DurableTaskBranch(this.deps.tenantId, {
      taskExternalId,
      nodeId,
      branchId,
    });
  }

  /**
   * The task external ID of a workflow run; for a run with several tasks, the first task's.
   * @param workflowRunId - The workflow run ID to look up.
   */
  async getTaskExternalId(workflowRunId: string): Promise<string> {
    const run = await this.get(workflowRunId);
    const tasks = run?.tasks;

    if (Array.isArray(tasks) && tasks.length > 0 && tasks[0]?.taskExternalId) {
      return tasks[0].taskExternalId;
    }

    throw new Error(`Could not find task external ID for workflow run ${workflowRunId}`);
  }
}

function metadataPairs(additionalMetadata: Record<string, string> | undefined): string[] {
  return Object.entries(additionalMetadata || {}).map(([key, value]) => `${key}:${value}`);
}
