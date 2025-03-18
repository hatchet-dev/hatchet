import * as z from 'zod';

import { CreateStep, CreateStepSchema } from './step';
import {
  ConcurrencyLimitStrategy as PbConcurrencyLimitStrategy,
  StickyStrategy as PbStickyStrategy,
} from './protoc/workflows';

const CronConfigSchema = z.object({
  cron: z.string(),
  event: z.undefined(),
});

const EventConfigSchema = z.object({
  cron: z.undefined(),
  event: z.string(),
});

const OnConfigSchema = z.union([CronConfigSchema, EventConfigSchema]).optional();

const StepsSchema = z.array(CreateStepSchema);

export type Steps = z.infer<typeof StepsSchema>;

export const ConcurrencyLimitStrategy = PbConcurrencyLimitStrategy;

export const WorkflowConcurrency = z.object({
  name: z.string(),
  maxRuns: z.number().optional(),
  limitStrategy: z.nativeEnum(ConcurrencyLimitStrategy).optional(),
  expression: z.string().optional(),
});

export const HatchetTimeoutSchema = z.string();

export const StickyStrategy = PbStickyStrategy;

export const CreateWorkflowSchema = z.object({
  id: z.string(),
  description: z.string(),
  version: z.string().optional(),
  /**
   * sticky will attempt to run all steps for workflow on the same worker
   */
  sticky: z.nativeEnum(StickyStrategy).optional(),
  scheduleTimeout: z.string().optional(),
  /**
   * @deprecated Workflow timeout is deprecated. Use step timeouts instead.
   */
  timeout: HatchetTimeoutSchema.optional(),
  on: OnConfigSchema,
  steps: StepsSchema,
  onFailure: CreateStepSchema?.optional(),
});

/**
 * @deprecated Use client.workflow instead (TODO link to migration doc)
 */
export interface Workflow extends z.infer<typeof CreateWorkflowSchema> {
  concurrency?: z.infer<typeof WorkflowConcurrency> & {
    key?: (ctx: any) => string;
  };
  steps: CreateStep<any, any>[];
  onFailure?: CreateStep<any, any>;
}
