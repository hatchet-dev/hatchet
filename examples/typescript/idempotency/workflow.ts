import { hatchet } from '../hatchet-client';

export type IdempotencyInput = {
  id: string;
};

// > idempotency
export const idempotentTask = hatchet.task<IdempotencyInput, { result: string }>({
  name: 'idempotent-task',
  idempotency: {
    expression: 'input.id',
    ttlMs: 60_000,
  },
  fn: async (input) => {
    return { result: `Hello, world from task ${input.id}` };
  },
});
