import { AdminClient } from '@hatchet/clients/admin';
import { Api } from '@hatchet/clients/rest';
import { CronWorkflows, CronWorkflowsList } from '@hatchet/clients/rest/generated/data-contracts';
import { z } from 'zod';
import { Workflow } from '@hatchet/workflow';
import { AxiosError } from 'axios';
import { ClientConfig } from '@hatchet/clients/hatchet-client/client-config';
import { Logger } from '@util/logger';

/**
 * Schema for creating a Cron Trigger.
 */
export const CreateCronTriggerSchema = z.object({
  name: z.string(),
  expression: z.string().refine((val) => {
    // Basic cron validation regex
    const cronRegex =
      /^(\*|([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])|\*\/([0-9]|1[0-9]|2[0-3])) (\*|([0-9]|1[0-9]|2[0-3])|\*\/([0-9]|1[0-9]|2[0-3])) (\*|([1-9]|1[0-9]|2[0-9]|3[0-1])|\*\/([1-9]|1[0-9]|2[0-9]|3[0-1])) (\*|([1-9]|1[0-2])|\*\/([1-9]|1[0-2])) (\*|([0-6])|\*\/([0-6]))$/;
    return cronRegex.test(val);
  }, 'Invalid cron expression'),
  input: z.record(z.any()).optional(),
  additionalMetadata: z.record(z.string()).optional(),
});

/**
 * Type representing the input for creating a Cron.
 */
export type CreateCronInput = z.infer<typeof CreateCronTriggerSchema>;

/**
 * Client for managing Cron Triggers.
 */
export class CronClient {
  private logger: Logger;

  /**
   * Initializes a new instance of CronClient.
   * @param tenantId - The tenant identifier.
   * @param config - Client configuration settings.
   * @param api - API instance for REST interactions.
   * @param adminClient - Admin client for administrative operations.
   */
  constructor(
    private readonly tenantId: string,
    private readonly config: ClientConfig,
    private readonly api: Api,
    private readonly adminClient: AdminClient
  ) {
    this.logger = config.logger('Cron', this.config.log_level);
  }

  /**
   * Retrieves the Cron ID from a CronWorkflows object or a string.
   * @param cron - The CronWorkflows object or Cron ID as a string.
   * @returns The Cron ID as a string.
   */
  private getCronId(cron: CronWorkflows | string): string {
    return typeof cron === 'string' ? cron : cron.metadata.id;
  }

  /**
   * Creates a new Cron workflow.
   * @param workflow - The workflow identifier or Workflow object.
   * @param cron - The input data for creating the Cron Trigger.
   * @returns A promise that resolves to the created CronWorkflows object.
   * @throws Will throw an error if the input is invalid or the API call fails.
   */
  async create(workflow: string | Workflow, cron: CreateCronInput): Promise<CronWorkflows> {
    const workflowId = typeof workflow === 'string' ? workflow : workflow.id;

    // Validate cron input with zod schema
    try {
      const parsedCron = CreateCronTriggerSchema.parse(cron);
      const response = await this.api.cronWorkflowTriggerCreate(this.tenantId, workflowId, {
        cronName: parsedCron.name,
        cronExpression: parsedCron.expression,
        input: parsedCron.input ?? {},
        additionalMetadata: parsedCron.additionalMetadata ?? {},
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
  async list(query: Parameters<typeof this.api.cronWorkflowList>[1]): Promise<CronWorkflowsList> {
    const response = await this.api.cronWorkflowList(this.tenantId, query);
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
