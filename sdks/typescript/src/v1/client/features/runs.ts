import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { V1TaskStatus, V1TaskFilter } from '@hatchet/clients/rest/generated/data-contracts';
import {
  RunEventType,
  RunListenerClient,
} from '@hatchet-dev/typescript-sdk/clients/listeners/run-listener/child-listener-client';
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
   * The parent task external id to filter by
   * @format uuid
   * @minLength 36
   * @maxLength 36
   */
  parentTaskExternalId?: string;

  /**
   * The triggering event external id to filter by
   * @format uuid
   * @minLength 36
   * @maxLength 36
   */
  triggeringEventExternalId?: string;
}

/**
 * RunsClient is used to list and manage runs
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
    this.listener = client.v0.listener;
  }

  async get<T = any>(run: string | WorkflowRunRef<T>) {
    const runId = typeof run === 'string' ? run : await run.getWorkflowRunId();

    const { data } = await this.api.v1WorkflowRunGet(runId);
    return data;
  }

  async get_status<T = any>(run: string | WorkflowRunRef<T>) {
    const runId = typeof run === 'string' ? run : await run.getWorkflowRunId();

    const { data } = await this.api.v1WorkflowRunGetStatus(runId);
    return data;
  }

  async list(opts?: Partial<ListRunsOpts>) {
    const { data } = await this.api.v1WorkflowRunList(this.tenantId, {
      ...(await this.prepareListFilter(opts || {})),
    });
    return data;
  }

  async cancel(opts: CancelRunOpts) {
    const filter = await this.prepareFilter(opts.filters || {});

    return this.api.v1TaskCancel(this.tenantId, {
      externalIds: opts.ids,
      filter: !opts.ids ? filter : undefined,
    });
  }

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
      triggering_event_external_id: opts.triggeringEventExternalId,
    };
  }

  runRef<T extends Record<string, any> = any>(id: string): WorkflowRunRef<T> {
    return new WorkflowRunRef<T>(id, this.listener, this);
  }

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
