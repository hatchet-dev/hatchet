import type { InputType } from '@hatchet/v1';
import { hatchet } from '../hatchet-client';

// Mirrors `sdks/python/examples/simple/worker.py` outputs for e2e.
export const helloWorld = hatchet.task({
  name: 'ts-hello-world',
  fn: async (_input: InputType) => {
    return { result: 'Hello, world!' };
  },
});

export const helloWorldDurable = hatchet.durableTask({
  name: 'ts-hello-world-durable',
  executionTimeout: '10m',
  fn: async (_input: InputType) => {
    return { result: 'Hello, world!' };
  },
});
