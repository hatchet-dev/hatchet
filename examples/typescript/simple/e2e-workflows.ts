import { hatchet } from '../hatchet-client';
import type { InputType } from '@hatchet-dev/typescript-sdk/v1';

// Mirrors `sdks/python/examples/simple/worker.py` outputs for e2e.
export const helloWorld = hatchet.task({
  name: 'hello-world',
  fn: async (_input: InputType) => {
    return { result: 'Hello, world!' };
  },
});

export const helloWorldDurable = hatchet.durableTask({
  name: 'hello-world-durable',
  executionTimeout: '10m',
  fn: async (_input: InputType) => {
    return { result: 'Hello, world!' };
  },
});

