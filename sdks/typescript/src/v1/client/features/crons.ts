import { CronWorkflows, CronWorkflowsList } from '@hatchet/clients/rest/generated/data-contracts';
import { z } from 'zod';
import { Workflow } from '@hatchet/workflow';
import { AxiosError } from 'axios';
import { isValidUUID } from '@util/uuid';
import { BaseWorkflowDeclaration } from '@hatchet/v1';
import { applyNamespace } from '@hatchet/util/apply-namespace';
import { HatchetClient } from '../client';
import { workflowNameString, WorkflowsClient } from './workflows';

/**
 * Validates a cron expression to support both 5-field and 6-field formats.
 * Also supports timezone prefixes like CRON_TZ= and TZ=.
 * @param expression - The cron expression to validate.
 * @returns True if valid, false otherwise.
 */
function validateCronExpression(expression: string): boolean {
  if (!expression || typeof expression !== 'string') {
    return false;
  }

  // Extract cron part for field count validation (timezone is supported but we validate the cron part)
  let cronPart = expression;
  if (expression.startsWith('CRON_TZ=')) {
    const parts = expression.split(' ', 2);
    if (parts.length === 2) {
      cronPart = parts[1];
    }
  } else if (expression.startsWith('TZ=')) {
    const parts = expression.split(' ', 2);
    if (parts.length === 2) {
      cronPart = parts[1];
    }
  }

  const fields = cronPart.trim().split(/\s+/);

  // Only allow 5 or 6 fields (not 7 with year field)
  return fields.length === 5 || fields.length === 6;
}

/**
 * Schema for creating a Cron Trigger.
 */
export const CreateCronTriggerSchema = z.object({
  name: z.string(),
  expression: z.string().refine(validateCronExpression, {
    message: 'Cron expression must have 5 fields (minute hour day month weekday) or 6 fields (second minute hour day month weekday)',
  }),
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
    const workflowId = applyNamespace(workflowNameString(workflow), this.namespace);

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
        applyNamespace(workflowNameString(workflow), this.namespace)
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
