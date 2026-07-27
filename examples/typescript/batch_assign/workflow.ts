import { hatchet } from '../hatchet-client';

// Mirrors `sdks/python/examples/batch_assign/worker.py` for e2e.
// Preview: batch tasks are in beta and may change in future releases.

type SimpleInput = { message: string };
type SimpleOutput = { transformed_message: string };

type KeyedInput = { message: string; group: string };
type KeyedFailableInput = { message: string; group: string | number };
type KeyedOutput = {
  batch_key?: string;
  batch_size?: number;
  unique_keys?: number;
  uppercase: string;
};

type LargePayloadInput = { data: string };
type LargeOutput = {
  batch_id: string;
  received: boolean;
  batch_size: number;
  data_length: number;
};

type SingleOutput = { original: string; batch_size: number };
type OrderedInput = { index: number };
type OrderedOutput = { index: number };
type BroadcastOutput = { sum: number };
type ChildOutput = { message_len: number };
type ChildBatchOutput = { out: Record<string, SimpleInput> };

export const batchSimple = hatchet.batchTask({
  name: 'batch-simple',
  batch: { maxSize: 3, maxInterval: 200 },
  fn: async (tasks: Record<string, SimpleInput>) => {
    const out: Record<string, SimpleOutput> = {};
    Object.entries(tasks).forEach(([id, input]) => {
      out[id] = { transformed_message: input.message.toUpperCase() };
    });
    return out;
  },
});

export const batchKeyed = hatchet.batchTask({
  name: 'batch-keyed',
  batch: { maxSize: 2, maxInterval: 200, groupKey: 'input.group' },
  fn: async (tasks: Record<string, KeyedInput>) => {
    const uniqueKeys = new Set(Object.values(tasks).map((i) => i.group)).size;
    const batchSize = Object.keys(tasks).length;
    const out: Record<string, KeyedOutput> = {};
    Object.entries(tasks).forEach(([id, input]) => {
      out[id] = {
        batch_key: input.group,
        batch_size: batchSize,
        unique_keys: uniqueKeys,
        uppercase: input.message.toUpperCase(),
      };
    });
    return out;
  },
});

export const batchKeyedFailable = hatchet.batchTask({
  name: 'batch-keyed-failable',
  batch: { maxSize: 2, maxInterval: 200, groupKey: 'input.group' },
  fn: async (tasks: Record<string, KeyedFailableInput>) => {
    const out: Record<string, KeyedOutput> = {};
    Object.entries(tasks).forEach(([id, input]) => {
      out[id] = { uppercase: input.message.toUpperCase() };
    });
    return out;
  },
});

export const batchKeyedInterval = hatchet.batchTask({
  name: 'batch-keyed-interval',
  batch: { maxSize: 3, maxInterval: 150, groupKey: 'input.group' },
  fn: async (tasks: Record<string, KeyedInput>) => {
    const uniqueKeys = new Set(Object.values(tasks).map((i) => i.group)).size;
    const batchSize = Object.keys(tasks).length;
    const out: Record<string, KeyedOutput> = {};
    Object.entries(tasks).forEach(([id, input]) => {
      out[id] = {
        batch_key: input.group,
        batch_size: batchSize,
        unique_keys: uniqueKeys,
        uppercase: input.message.toUpperCase(),
      };
    });
    return out;
  },
});

export const batchLarge = hatchet.batchTask({
  name: 'batch-large',
  batch: { maxSize: 100, maxInterval: 10_000 },
  fn: async (tasks: Record<string, LargePayloadInput>) => {
    const batchId = crypto.randomUUID();
    const batchSize = Object.keys(tasks).length;
    const out: Record<string, LargeOutput> = {};
    Object.entries(tasks).forEach(([id, input]) => {
      out[id] = {
        batch_id: batchId,
        received: true,
        batch_size: batchSize,
        data_length: input.data.length,
      };
    });
    return out;
  },
});

export const batchSingle = hatchet.batchTask({
  name: 'batch-single',
  batch: { maxSize: 1, maxInterval: 100 },
  fn: async (tasks: Record<string, SimpleInput>) => {
    const batchSize = Object.keys(tasks).length;
    const out: Record<string, SingleOutput> = {};
    Object.entries(tasks).forEach(([id, input]) => {
      out[id] = { original: input.message, batch_size: batchSize };
    });
    return out;
  },
});

export const batchOrdered = hatchet.batchTask({
  name: 'batch-ordered',
  batch: { maxSize: 20, maxInterval: 2_000 },
  fn: async (tasks: Record<string, OrderedInput>) => {
    const out: Record<string, OrderedOutput> = {};
    Object.entries(tasks).forEach(([id, input]) => {
      out[id] = { index: input.index };
    });
    return out;
  },
});

export const batchBroadcast = hatchet.batchTask({
  name: 'batch-broadcast',
  batch: { maxSize: 10, maxInterval: 2_000, broadcastOutput: true },
  fn: async (tasks: Record<string, SimpleInput>): Promise<BroadcastOutput> => {
    const sum = Object.values(tasks).reduce((acc, i) => acc + i.message.length, 0);
    return { sum };
  },
});

export const batchCancel = hatchet.batchTask({
  name: 'batch-cancel',
  batch: { maxSize: 10, maxInterval: 2_000, broadcastOutput: true },
  fn: async (_tasks: Record<string, SimpleInput>, ctx) => {
    await ctx.cancel();
    return {};
  },
});

export const child = hatchet.task({
  name: 'batch-child',
  fn: async (input: SimpleInput): Promise<ChildOutput> => ({ message_len: input.message.length }),
});

export const childBatch = hatchet.batchTask({
  name: 'batch-child-batch',
  batch: { maxSize: 10, maxInterval: 60_000, broadcastOutput: true },
  fn: async (tasks: Record<string, SimpleInput>): Promise<ChildBatchOutput> => ({ out: tasks }),
});

export const batchChildSpawn = hatchet.batchTask({
  name: 'batch-child-spawn',
  batch: { maxSize: 10, maxInterval: 60_000 },
  executionTimeout: '60s',
  fn: async (tasks: Record<string, SimpleInput>) => {
    const out: Record<string, ChildOutput> = {};
    await Promise.all(
      Object.keys(tasks).map(async (id) => {
        out[id] = await child.run({ message: 'blahblah' });
      })
    );
    return out;
  },
});

export const batchChildBatchSpawn = hatchet.batchTask({
  name: 'batch-child-batch-spawn',
  batch: { maxSize: 10, maxInterval: 60_000 },
  executionTimeout: '60s',
  fn: async (tasks: Record<string, SimpleInput>) => {
    const out: Record<string, ChildBatchOutput> = {};
    await Promise.all(
      Object.keys(tasks).map(async (id) => {
        out[id] = await childBatch.run({ message: 'hello' });
      })
    );
    return out;
  },
});
