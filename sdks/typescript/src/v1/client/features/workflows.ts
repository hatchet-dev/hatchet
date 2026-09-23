import { BaseWorkflowDeclaration, WorkflowDefinition } from '@hatchet/v1';
import type { LegacyWorkflow } from '@hatchet/legacy/legacy-transformer';
import { isLegacyWorkflow, warnLegacyWorkflow } from '@hatchet/legacy/legacy-transformer';
import { isValidUUID } from '@util/uuid';
import {
  WorkflowPauseScheduledCronRunQueueBehavior,
  WorkflowUpdateRequest,
} from '@hatchet/clients/rest/generated/data-contracts';
import { HatchetClient } from '../client';
import { Duration, durationToString } from '../duration';

export { WorkflowPauseScheduledCronRunQueueBehavior };

export type PauseWorkflowOpts = {
  /** How long runs stay queued while the workflow is paused before they are dropped. */
  queueTTL: Duration;
  /** The behavior of cron runs triggered while the workflow is paused. Defaults to QUEUE. */
  pausedWorkflowCronRunQueueBehavior?: WorkflowPauseScheduledCronRunQueueBehavior;
  /** The behavior of scheduled runs triggered while the workflow is paused. Defaults to QUEUE. */
  pausedWorkflowScheduledRunQueueBehavior?: WorkflowPauseScheduledCronRunQueueBehavior;
};

export const workflowNameString = (
  workflow: string | WorkflowDefinition | BaseWorkflowDeclaration<any, any> | LegacyWorkflow
) => {
  if (typeof workflow === 'string') {
    return workflow;
  }
  if (typeof workflow === 'object' && 'name' in workflow) {
    return workflow.name as string;
  }
  if (isLegacyWorkflow(workflow)) {
    warnLegacyWorkflow();
    return workflow.id;
  }
  throw new Error(
    'Invalid workflow: must be a string, Workflow object, or WorkflowDefinition object'
  );
};

/**
 * The workflows client is a client for managing workflows programmatically within Hatchet.
 *
 * NOTE: that workflows are the declaration, not the individual runs. If you're looking for runs, use the RunsClient instead.
 *
 */
export class WorkflowsClient {
  api: HatchetClient['api'];
  tenantId: string;
  // Add a cache for workflows
  private workflowCache: Map<string, { workflow: any; expiry: number }> = new Map();
  // Default cache TTL: 5 minutes (in ms)
  private cacheTTL: number = 5 * 60 * 1000;

  constructor(client: HatchetClient, cacheTTL?: number) {
    this.api = client.api;
    this.tenantId = client.tenantId;
    // Allow custom cache TTL if provided
    if (cacheTTL !== undefined) {
      this.cacheTTL = cacheTTL;
    }
  }

  /**
   * Gets the workflow ID from a workflow name, ID, or object.
   * If the input is not a valid UUID, it will look up the workflow by name.
   * @param workflow - The workflow name, ID, or object.
   * @returns The workflow ID as a string.
   */
  async getWorkflowIdFromName(
    workflow: string | WorkflowDefinition | BaseWorkflowDeclaration<any, any> | LegacyWorkflow
  ): Promise<string> {
    const str = (() => {
      if (typeof workflow === 'string') {
        return workflow;
      }

      if (typeof workflow === 'object' && 'name' in workflow) {
        return workflow.name;
      }

      throw new Error(
        'Invalid workflow: must be a string, Workflow object, or WorkflowDefinition object'
      );
    })();

    if (!isValidUUID(str)) {
      const wf = await this.get(str);
      if (!wf) {
        throw new Error('Invalid workflow ID: must be a valid UUID');
      }
      return wf.metadata.id;
    }

    return str;
  }

  /**
   * Get a workflow by its name, ID, or object.
   * @param workflow - The workflow name, ID, or object.
   * @returns A promise that resolves to the workflow.
   */
  async get(workflow: string | BaseWorkflowDeclaration<any, any> | LegacyWorkflow) {
    // Get workflow name string
    const name = workflowNameString(workflow);

    // Check cache first
    const cached = this.workflowCache.get(name);
    if (cached && cached.expiry > Date.now()) {
      return cached.workflow as NonNullable<
        Awaited<ReturnType<typeof this.api.workflowList>>['data']['rows']
      >[number];
    }

    // If not in cache or expired, fetch from API
    try {
      // Since the API only supports listing with a name filter,
      // we'll use the list endpoint with a name filter
      const { data } = await this.api.workflowList(this.tenantId, {
        name,
      });

      if (data && data.rows && data.rows.length > 0) {
        const wf = data.rows.find((row) => row.name === name) ?? data.rows[0];

        this.workflowCache.set(name, {
          workflow: wf,
          expiry: Date.now() + this.cacheTTL,
        });

        return wf;
      }

      throw new Error(`Workflow with name ${name} not found`);
    } catch (error) {
      // Clear cache on error
      this.workflowCache.delete(name);
      throw error;
    }
  }

  /**
   * List all workflows in the tenant.
   * @param opts - The options for the list operation.
   * @returns A promise that resolves to the list of workflows.
   */
  async list(opts?: Parameters<typeof this.api.workflowList>[1]) {
    const { data } = await this.api.workflowList(this.tenantId, opts);
    return data;
  }

  /**
   * Delete a workflow by its name, ID, or object.
   * @param workflow - The workflow name, ID, or object.
   * @returns A promise that resolves to the deleted workflow.
   */
  async delete(workflow: string | BaseWorkflowDeclaration<any, any> | LegacyWorkflow) {
    const name = workflowNameString(workflow);

    try {
      // Get the workflow first to find its ID
      const workflowObj = await this.get(name);

      if (!workflowObj || !workflowObj.metadata || !workflowObj.metadata.id) {
        throw new Error(`Could not find workflow with name ${name}`);
      }

      const { data } = await this.api.workflowDelete(workflowObj.metadata.id);

      // Remove from cache after deletion
      this.workflowCache.delete(name);

      return data;
    } catch (error) {
      // Clear cache on error
      this.workflowCache.delete(name);
      throw error;
    }
  }

  /**
   * Pause a workflow. While paused, new runs of the workflow are queued but not started.
   * @param workflow - The workflow name, ID, or object.
   * @param opts - The options for the pause operation.
   * @returns A promise that resolves to the updated workflow.
   */
  async pause(
    workflow: string | BaseWorkflowDeclaration<any, any> | LegacyWorkflow,
    opts: PauseWorkflowOpts
  ) {
    return this.update(workflow, {
      pause: {
        action: 'pause',
        pausedWorkflowQueueTTL: durationToString(opts.queueTTL),
        pausedWorkflowCronRunQueueBehavior:
          opts.pausedWorkflowCronRunQueueBehavior ??
          WorkflowPauseScheduledCronRunQueueBehavior.QUEUE,
        pausedWorkflowScheduledRunQueueBehavior:
          opts.pausedWorkflowScheduledRunQueueBehavior ??
          WorkflowPauseScheduledCronRunQueueBehavior.QUEUE,
      },
    });
  }

  /**
   * Unpause a workflow.
   * @param workflow - The workflow name, ID, or object.
   * @returns A promise that resolves to the updated workflow.
   */
  async unpause(workflow: string | BaseWorkflowDeclaration<any, any> | LegacyWorkflow) {
    return this.update(workflow, { pause: { action: 'unpause' } });
  }

  private async update(
    workflow: string | BaseWorkflowDeclaration<any, any> | LegacyWorkflow,
    request: WorkflowUpdateRequest
  ) {
    const name = workflowNameString(workflow);
    const workflowId = await this.getWorkflowIdFromName(name);

    try {
      const { data } = await this.api.workflowUpdate(workflowId, request);
      this.workflowCache.delete(name);
      this.workflowCache.delete(data.name);
      return data;
    } catch (error) {
      this.workflowCache.delete(name);
      throw error;
    }
  }
}
