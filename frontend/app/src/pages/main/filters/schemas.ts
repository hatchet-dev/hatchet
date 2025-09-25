import { z } from 'zod';

export const createFilterSchema = z.object({
  workflowId: z
    .string()
    .min(36, 'Workflow ID must be 36 characters')
    .max(36, 'Workflow ID must be 36 characters'),
  expression: z.string().min(1, 'Expression is required'),
  scope: z.string().min(1, 'Scope is required'),
  payload: z.string().default(''),
});

export const updateFilterSchema = z.object({
  expression: z.string().min(1, 'Expression is required').optional(),
  scope: z.string().min(1, 'Scope is required').optional(),
  payload: z.string().optional(),
});

export type CreateFilterFormData = z.infer<typeof createFilterSchema>;
export type UpdateFilterFormData = z.infer<typeof updateFilterSchema>;
