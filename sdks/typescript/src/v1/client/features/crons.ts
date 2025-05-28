import { CronWorkflows, CronWorkflowsList } from '@hatchet/clients/rest/generated/data-contracts';
import { z } from 'zod';
import { Workflow } from '@hatchet/workflow';
import { AxiosError } from 'axios';
import { isValidUUID } from '@util/uuid';
import { BaseWorkflowDeclaration } from '@hatchet/v1/declaration';
import { withNamespace } from '@hatchet/util/with-namespace';
import { HatchetClient } from '../client';
import { workflowNameString, WorkflowsClient } from './workflows';

/**
 * Schema for creating a Cron Trigger.
 */
export const CreateCronTriggerSchema = z.object({
  name: z.string(),
  expression: z.string(),
  input: z.record(z.any()).optional(),
  additionalMetadata: z.record(z.string()).optional(),
  priority: z.number().optional(),
});

/**
 * Type representing the input for creating a Cron.
 */
export type CreateCronInput = z.infer<typeof CreateCronTriggerSchema>;

/**
 * Client for managing Cron Triggers.
 */
export class CronClient {
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
   * Retrieves the Cron ID from a CronWorkflows object or a string.
   * @param cron - The CronWorkflows object or Cron ID as a string.
   * @returns The Cron ID as a string.
   */
  private getCronId(cron: CronWorkflows | string): string {
    const str = typeof cron === 'string' ? cron : cron.metadata.id;

    if (!isValidUUID(str)) {
      throw new Error('Invalid cron ID: must be a valid UUID');
    }

    return str;
  }

  /**
   * Creates a new Cron workflow.
   * @param workflow - The workflow identifier or Workflow object.
   * @param cron - The input data for creating the Cron Trigger.
   * @returns A promise that resolves to the created CronWorkflows object.
   * @throws Will throw an error if the input is invalid or the API call fails.
   */
  async create(
    workflow: string | Workflow | BaseWorkflowDeclaration<any, any>,
    cron: CreateCronInput
  ): Promise<CronWorkflows> {
    const workflowId = withNamespace(workflowNameString(workflow), this.namespace);

    // Validate cron input with zod schema
    try {
      const parsedCron = CreateCronTriggerSchema.parse(cron);
      const response = await this.api.cronWorkflowTriggerCreate(this.tenantId, workflowId, {
        cronName: parsedCron.name,
        cronExpression: parsedCron.expression,
        input: parsedCron.input ?? {},
        additionalMetadata: parsedCron.additionalMetadata ?? {},
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
   * Deletes an existing Cron Trigger.
   * @param cron - The Cron Trigger ID as a string or CronWorkflows object.
   * @returns A promise that resolves when the Cron Trigger is deleted.
   */
  async delete(cron: string | CronWorkflows): Promise<void> {
    const cronId = this.getCronId(cron);
    await this.api.workflowCronDelete(this.tenantId, cronId);
  }

  /**
   * Lists all Cron Triggers based on the provided query parameters.
   * @param query - Query parameters for listing Cron Triggers.
   * @returns A promise that resolves to a CronWorkflowsList object.
   */
  async list(
    query: Parameters<typeof this.api.cronWorkflowList>[1] & {
      workflow?: string | Workflow | BaseWorkflowDeclaration<any, any>;
    }
  ): Promise<CronWorkflowsList> {
    const { workflow, ...rest } = query;

    if (workflow) {
      const workflowId = await this.workflows.getWorkflowIdFromName(
        withNamespace(workflowNameString(workflow), this.namespace)
      );
      rest.workflowId = workflowId;
    }

    const response = await this.api.cronWorkflowList(this.tenantId, rest);
    return response.data;
  }

  /**
   * Retrieves a specific Cron Trigger by its ID.
   * @param cron - The Cron Trigger ID as a string or CronWorkflows object.
   * @returns A promise that resolves to the CronWorkflows object.
   */
  async get(cron: string | CronWorkflows): Promise<CronWorkflows> {
    const cronId = this.getCronId(cron);
    const response = await this.api.workflowCronGet(this.tenantId, cronId);
    return response.data;
  }
}
