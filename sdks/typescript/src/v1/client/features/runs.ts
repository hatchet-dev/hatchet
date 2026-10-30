import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import {
  RunEventType,
  RunListenerClient,
} from '@hatchet/clients/listeners/run-listener/child-listener-client';
import { WorkflowsClient } from './workflows';
import { HatchetClient } from '../client';
import { createV1AdminRpc } from '@hatchet/clients/admin/rpc';
import type { AdminServiceClient } from '@hatchet/protoc/v1/workflows';
import type { Transport } from '@hatchet/clients/transport/transport';
import { runStatusToJSON, toRunDetail } from '@hatchet/core/run-detail';
import type { RunDetail, TaskRunDetail } from '@hatchet/core/types';
import { RestRunsClient, type ListRunsOpts, type RunFilter } from '@hatchet/clients/rest/runs';

export type { RunDetail, TaskRunDetail, ListRunsOpts, RunFilter };

// Keep runStatusToJSON importable for callers who want the raw proto status as a string.
export { runStatusToJSON };

export type CancelRunOpts = {
  ids?: string[];
  filters?: RunFilter;
};

export type ReplayRunOpts = {
  ids?: string[];
  filters?: RunFilter;
};

/**
 * The runs client is a client for interacting with task and workflow runs within Hatchet.
 */
export class RunsClient {
  api: HatchetClient['api'];
  tenantId: string;
  workflows: WorkflowsClient;
  listener: RunListenerClient;
  private _transport: Transport;
  private _adminRpc: AdminServiceClient | undefined;
  private rest: RestRunsClient;

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.tenantId = client.tenantId;
    this.workflows = client.workflows;
    this.listener = client._listener;
    this._transport = client.transport;
    this.rest = new RestRunsClient({
      api: client.api,
      tenantId: client.tenantId,
      workflows: client.workflows,
    });
  }

  /** The v1 admin RPCs on the client's Connect transport, the same path the core client uses. */
  private get adminRpc(): AdminServiceClient {
    if (!this._adminRpc) {
      this._adminRpc = createV1AdminRpc(this._transport);
    }
    return this._adminRpc;
  }

  /**
   * Gets a task or workflow run by its ID.
   * @param run - The ID of the run to get.
   * @returns A promise that resolves to the run.
   */
  async get<T = any>(run: string | WorkflowRunRef<T>) {
    const runId = typeof run === 'string' ? run : await run.getWorkflowRunId();
    return this.rest.get(runId);
  }

  /**
   * Gets the status of a task or workflow run by its ID.
   * @param run - The ID of the run to get the status of.
   * @returns A promise that resolves to the status of the run.
   */
  async get_status<T = any>(run: string | WorkflowRunRef<T>) {
    const runId = typeof run === 'string' ? run : await run.getWorkflowRunId();
    return this.rest.getStatus(runId);
  }

  /**
   * Gets run details
   * @param run - The workflow run ID (string) or a WorkflowRunRef.
   * @returns A promise resolving to GetRunDetailsResponse with task run statuses and outputs.
   */
  async getDetails<T = any>(run: string | WorkflowRunRef<T>): Promise<RunDetail> {
    const runId = typeof run === 'string' ? run : await run.getWorkflowRunId();
    return toRunDetail(await this.adminRpc.getRunDetails({ externalId: runId }));
  }

  /**
   * Lists all task and workflow runs for the current tenant.
   * @param opts - The options for the list operation.
   * @returns A promise that resolves to the list of runs.
   */
  async list(opts?: Partial<ListRunsOpts>) {
    return this.rest.list(opts);
  }

  /**
   * Cancels a task or workflow run by its ID.
   * @param opts - The options for the cancel operation.
   * @returns A promise that resolves to the cancelled run.
   */
  async cancel(opts: CancelRunOpts) {
    const filter = await this.rest.prepareFilter(opts.filters || {});

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
    const filter = await this.rest.prepareFilter(opts.filters || {});
    return this.api.v1TaskReplay(this.tenantId, {
      externalIds: opts.ids,
      filter: !opts.ids ? filter : undefined,
    });
  }

  /**
   * Restore an evicted durable task so it can resume execution.
   * @param taskExternalId - The external ID of the evicted task.
   */
  async restoreTask(taskExternalId: string) {
    return this.rest.restoreTask(taskExternalId);
  }

  /**
   * Fork (reset) a durable task from a specific node, triggering re-execution from that point.
   * @param taskExternalId - The external ID of the durable task to reset.
   * @param nodeId - The node ID to replay from.
   */
  async branchDurableTask(taskExternalId: string, nodeId: number, branchId: number = 0) {
    return this.rest.branchDurableTask(taskExternalId, nodeId, branchId);
  }

  /**
   * Resolve the task external ID for a workflow run. For runs with multiple tasks,
   * returns the first task's external ID.
   * @param workflowRunId - The workflow run ID to look up.
   * @returns The task external ID.
   */
  async getTaskExternalId(workflowRunId: string): Promise<string> {
    return this.rest.getTaskExternalId(workflowRunId);
  }

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
