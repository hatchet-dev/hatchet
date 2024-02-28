import * as z from 'zod';

import { CreateStep, CreateStepSchema } from './step';
import { ConcurrencyLimitStrategy as PbConcurrencyLimitStrategy } from './protoc/workflows';

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

export const ConcurrencyLimitStrategy = PbConcurrencyLimitStrategy;

export const WorkflowConcurrency = z.object({
  name: z.string(),
  maxRuns: z.number().optional(),
  limitStrategy: z.nativeEnum(ConcurrencyLimitStrategy).optional(),
});

export const HatchetTimeoutSchema = z.string();

export const CreateWorkflowSchema = z.object({
  id: z.string(),
  description: z.string(),
  version: z.string().optional(),
  scheduleTimeout: z.string().optional(),
  timeout: HatchetTimeoutSchema.optional(),
  on: OnConfigSchema,
  steps: StepsSchema,
});

export interface Workflow extends z.infer<typeof CreateWorkflowSchema> {
  concurrency?: z.infer<typeof WorkflowConcurrency> & {
    key: (ctx: any) => string;
  };
  steps: CreateStep<any, any>[];
}
