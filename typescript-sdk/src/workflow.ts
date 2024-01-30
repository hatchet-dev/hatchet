import * as z from 'zod';

import { CreateStep, CreateStepSchema } from './step';

const CronConfigSchema = z.object({
  cron: z.string(),
  event: z.undefined(),
});

const EventConfigSchema = z.object({
  cron: z.undefined(),
  event: z.string(),
});

const OnConfigSchema = z.union([CronConfigSchema, EventConfigSchema]);

const StepsSchema = z.array(CreateStepSchema);

export type Steps = z.infer<typeof StepsSchema>;

export const CreateWorkflowSchema = z.object({
  id: z.string(),
  description: z.string(),
  on: OnConfigSchema,
  steps: StepsSchema,
});

export interface Workflow extends z.infer<typeof CreateWorkflowSchema> {
  steps: CreateStep<any>[];
}
