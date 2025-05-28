import {
  ScheduledWorkflows,
  ScheduledWorkflowsList,
} from '@hatchet/clients/rest/generated/data-contracts';
import { z } from 'zod';
import { Workflow } from '@hatchet/workflow';
import { AxiosError } from 'axios';
import { isValidUUID } from '@util/uuid';
import { BaseWorkflowDeclaration, WorkflowDefinition } from '@hatchet/v1';
import { applyNamespace } from '@hatchet/util/apply-namespace';
import { HatchetClient } from '../client';
import { workflowNameString, WorkflowsClient } from './workflows';
/**
 * Schema for creating a Scheduled Run Trigger.
 */
export const CreateScheduledRunTriggerSchema = z.object({
  triggerAt: z.coerce.date(),
  input: z.record(z.any()).optional(),
  additionalMetadata: z.record(z.string()).optional(),
  priority: z.number().optional(),
});

/**
 * Type representing the input for creating a Cron.
 */
export type CreateScheduledRunInput = z.infer<typeof CreateScheduledRunTriggerSchema>;

/**
 * Client for managing Scheduled Runs.
 */
export class ScheduleClient {
  api: HatchetClient['api'];
  tenantId: string;
  workflows: WorkflowsClient;
  namespace: string | undefined;

  constructor(client: HatchetClient) {
    this.api = client.api;
    this.tenantId = client.tenantId;
    this.workflows = new WorkflowsClient(client);
    this.namespace = client.config.namespace;
  }

  /**
   * Retrieves the Scheduled Run ID from a ScheduledRun object or a string.
   * @param scheduledRun - The ScheduledRun object or Scheduled Run ID as a string.
   * @returns The Scheduled Run ID as a string.
   */
  private getScheduledRunId(scheduledRun: ScheduledWorkflows | string): string {
    const str = typeof scheduledRun === 'string' ? scheduledRun : scheduledRun.metadata.id;

    if (!isValidUUID(str)) {
      throw new Error('Invalid scheduled run ID: must be a valid UUID');
    }

    return str;
  }

  /**
   * Creates a new Scheduled Run.
   * @param workflow - The workflow name or Workflow object.
   * @param scheduledRun - The input data for creating the Scheduled Run.
   * @returns A promise that resolves to the created ScheduledWorkflows object.
   * @throws Will throw an error if the input is invalid or the API call fails.
   */
  async create(
    workflow: string | Workflow,
    cron: CreateScheduledRunInput
  ): Promise<ScheduledWorkflows> {
    const workflowId = applyNamespace(workflowNameString(workflow), this.namespace);

    // Validate cron input with zod schema
    try {
      const parsedCron = CreateScheduledRunTriggerSchema.parse(cron);

      const response = await this.api.scheduledWorkflowRunCreate(this.tenantId, workflowId, {
        input: parsedCron.input ?? {},
        additionalMetadata: parsedCron.additionalMetadata ?? {},
        triggerAt: parsedCron.triggerAt.toISOString(),
        priority: parsedCron.priority,
      });
      return response.data;
    } catch (err) {
      if (err instanceof z.ZodError) {
        throw new Error(`Invalid cron input: ${err.message}`);
      }

      if (err instanceof AxiosError) {
        throw new Error(JSON.stringify(err.response?.data.errors));
      }

      throw err;
    }
  }

  /**
   * Deletes an existing Scheduled Run.
   * @param scheduledRun - The Scheduled Run ID as a string or ScheduledWorkflows object.
   * @returns A promise that resolves when the Scheduled Run is deleted.
   */
  async delete(scheduledRun: string | ScheduledWorkflows): Promise<void> {
    const scheduledRunId = this.getScheduledRunId(scheduledRun);
    await this.api.workflowScheduledDelete(this.tenantId, scheduledRunId);
  }

  /**
   * Lists all Cron Triggers based on the provided query parameters.
   * @param query - Query parameters for listing Scheduled Runs.
   * @returns A promise that resolves to a ScheduledWorkflowsList object.
   */
  async list(
    query: Parameters<typeof this.api.workflowScheduledList>[1] & {
      workflow?: string | Workflow | WorkflowDefinition | BaseWorkflowDeclaration<any, any>;
    }
  ): Promise<ScheduledWorkflowsList> {
    const { workflow, ...rest } = query;

    if (workflow) {
      const workflowId = await this.workflows.getWorkflowIdFromName(
        applyNamespace(workflowNameString(workflow), this.namespace)
      );
      rest.workflowId = workflowId;
    }

    const response = await this.api.workflowScheduledList(this.tenantId, rest);
    return response.data;
  }

  /**
   * Retrieves a specific Scheduled Run by its ID.
   * @param scheduledRun - The Scheduled Run ID as a string or ScheduledWorkflows object.
   * @returns A promise that resolves to the ScheduledWorkflows object.
   */
  async get(scheduledRun: string | ScheduledWorkflows): Promise<ScheduledWorkflows> {
    const scheduledRunId = this.getScheduledRunId(scheduledRun);
    const response = await this.api.workflowScheduledGet(this.tenantId, scheduledRunId);
    return response.data;
  }
}
