import { hatchet } from '../hatchet-client';

export const EVENT_KEY = 'ts-e2e:idempotency-example';

export type IdempotencyInput = {
  id: string;
};

// > idempotency
export const idempotentTask = hatchet.task<IdempotencyInput, { result: string }>({
  name: 'ts-e2e-idempotent-task',
  idempotency: {
    expression: 'input.id',
    ttlMs: 60_000,
  },
  onEvents: [EVENT_KEY],
  fn: async (input) => {
    return { result: `Hello, world from task ${input.id}` };
  },
});
// !!

export const idempotentTaskShortWindow = hatchet.task<IdempotencyInput, { result: string }>({
  name: 'ts-e2e-idempotent-task-short-window',
  idempotency: {
    expression: 'input.id',
    ttlMs: 2_000,
  },
  fn: async (input) => {
    return { result: `Hello, world from task ${input.id}` };
  },
});
