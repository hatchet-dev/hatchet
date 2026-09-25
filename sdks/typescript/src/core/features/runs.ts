import type { AdminServiceClient, TasksFilter } from '@hatchet/protoc/v1/workflows';
import { toRunDetail } from '../run-detail';
import { WorkflowRunRef } from '../run-ref';
import type { CallOptions, CancelRunOpts, ReplayRunOpts, RunDetail, RunFilter } from '../types';

/**
 * Task and workflow runs, over the v1 `AdminService`. Everything here is a unary call, so
 * it works from any runtime the client runs on; listing runs is a REST feature of the Node
 * client and is not available here.
 */
export class RunsClient {
  constructor(private readonly rpc: AdminServiceClient) {}

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
    return toRunDetail(await this.rpc.getRunDetails({ externalId }, options));
  }

  /**
   * Cancels runs by id, or every run matching the filter when no ids are given.
   * @returns the ids of the cancelled tasks
   */
  async cancel(opts: CancelRunOpts) {
    return this.rpc.cancelTasks({
      externalIds: opts.ids ?? [],
      filter: opts.ids ? undefined : prepareFilter(opts.filters ?? {}),
    });
  }

  /**
   * Replays runs by id, or every run matching the filter when no ids are given.
   * @returns the ids of the replayed tasks
   */
  async replay(opts: ReplayRunOpts) {
    return this.rpc.replayTasks({
      externalIds: opts.ids ?? [],
      filter: opts.ids ? undefined : prepareFilter(opts.filters ?? {}),
    });
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
