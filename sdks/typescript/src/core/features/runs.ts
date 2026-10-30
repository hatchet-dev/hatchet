import type { AdminServiceClient, TasksFilter } from '@hatchet/protoc/v1/workflows';
import type { ListRunsOpts, RestRunsClient } from '@hatchet/clients/rest/runs';
import { toRunDetail } from '../run-detail';
import { WorkflowRunRef } from '../run-ref';
import type { CallOptions, CancelRunOpts, ReplayRunOpts, RunDetail, RunFilter } from '../types';

export interface RunsClientDeps {
  rpc: AdminServiceClient;
  /** The run operations that go over the REST API. */
  rest: RestRunsClient;
}

/**
 * Task and workflow runs. Getting, cancelling and replaying a run go over the v1
 * `AdminService`; listing runs and restoring or branching durable tasks go over the REST
 * API, with the Node client's names and signatures.
 */
export class RunsClient {
  constructor(private readonly deps: RunsClientDeps) {}

  /**
   * Gets a run's state: its status, whether it is done, its input and metadata, and each of
   * its tasks with their status, output and error.
   * @param run - The run id or a reference to the run.
   * @param options - A signal or deadline for the call.
   */
  async get<T = unknown>(
    run: string | WorkflowRunRef<T>,
    options?: CallOptions
  ): Promise<RunDetail> {
    return this.getDetails(run, options);
  }

  /** @alias get */
  async getDetails<T = unknown>(
    run: string | WorkflowRunRef<T>,
    options?: CallOptions
  ): Promise<RunDetail> {
    const externalId = typeof run === 'string' ? run : await run.getWorkflowRunId();
    return toRunDetail(await this.deps.rpc.getRunDetails({ externalId }, options));
  }

  /**
   * Lists the tenant's task and workflow runs matching the options. Workflow names are
   * resolved to ids through the REST API first.
   * @param opts - The options for the list operation.
   * @returns the matching runs
   */
  async list(opts?: Partial<ListRunsOpts>) {
    return this.deps.rest.list(opts);
  }

  /**
   * Cancels runs by id, or every run matching the filter when no ids are given.
   * @returns the ids of the cancelled tasks
   */
  async cancel(opts: CancelRunOpts) {
    return this.deps.rpc.cancelTasks({
      externalIds: opts.ids ?? [],
      filter: opts.ids ? undefined : prepareFilter(opts.filters ?? {}),
    });
  }

  /**
   * Replays runs by id, or every run matching the filter when no ids are given.
   * @returns the ids of the replayed tasks
   */
  async replay(opts: ReplayRunOpts) {
    return this.deps.rpc.replayTasks({
      externalIds: opts.ids ?? [],
      filter: opts.ids ? undefined : prepareFilter(opts.filters ?? {}),
    });
  }

  /**
   * Restores an evicted durable task so it can resume execution.
   * @param taskExternalId - The external ID of the evicted task.
   */
  async restoreTask(taskExternalId: string) {
    return this.deps.rest.restoreTask(taskExternalId);
  }

  /**
   * Forks (resets) a durable task from a specific node, triggering re-execution from that point.
   * @param taskExternalId - The external ID of the durable task to reset.
   * @param nodeId - The node ID to replay from.
   */
  async branchDurableTask(taskExternalId: string, nodeId: number, branchId: number = 0) {
    return this.deps.rest.branchDurableTask(taskExternalId, nodeId, branchId);
  }

  /** A reference to an existing run, to wait on, cancel or replay it. */
  runRef<T = unknown>(id: string): WorkflowRunRef<T> {
    return new WorkflowRunRef<T>(id, this);
  }
}

/** Builds the proto filter; `since` defaults to one hour ago, as the Node client's does. */
function prepareFilter(filter: RunFilter): TasksFilter {
  return {
    since: filter.since ?? new Date(Date.now() - 60 * 60 * 1000),
    until: filter.until,
    statuses: filter.statuses ?? [],
    workflowIds: filter.workflowIds ?? [],
    additionalMetadata: Object.entries(filter.additionalMetadata ?? {}).map(
      ([key, value]) => `${key}:${value}`
    ),
  };
}
