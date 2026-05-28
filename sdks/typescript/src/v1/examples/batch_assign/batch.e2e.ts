import { randomUUID } from 'crypto';
import sleep from '@hatchet/util/sleep';
import { hatchet } from '../hatchet-client';
import { Worker } from '../../client/worker/worker';
import type { Context } from '../../client/worker/context';

describe('batch-task e2e', () => {
  jest.setTimeout(1200000);

  let worker: Worker;
  const runId = randomUUID();

  // Test 1-2: simple non-keyed batch
  const batchWorkflow = hatchet.batchTask({
    name: `batch-e2e-simple-${runId}`,
    retries: 0,
    batchMaxSize: 3,
    batchMaxInterval: '200ms',
    fn: (tasks: Array<readonly [{ Message: string }, Context<{ Message: string }>]>) =>
      tasks.map(([input]) => ({
        TransformedMessage: input.Message.toUpperCase(),
      })),
  });

  // Test 3: keyed batch, partition by key when batch size reached
  const keyedBatchWorkflow = hatchet.batchTask<
    { Message: string; group: string },
    { batchKey: string; batchSize: number; uniqueKeys: number; uppercase: string }
  >({
    name: `batch-e2e-keyed-${runId}`,
    retries: 0,
    batchMaxSize: 2,
    batchMaxInterval: '200ms',
    batchGroupKey: 'input.group',
    fn: (tasks) =>
      tasks.map(([input]) => ({
        batchKey: input.group,
        batchSize: tasks.length,
        uniqueKeys: new Set(tasks.map(([item]) => item.group)).size,
        uppercase: input.Message.toUpperCase(),
      })),
  });

  // Test 4: keyed batch, flush independently on interval
  const keyedIntervalWorkflow = hatchet.batchTask<
    { Message: string; group: string },
    { batchKey: string; batchSize: number; uniqueKeys: number; payload: string }
  >({
    name: `batch-e2e-keyed-interval-${runId}`,
    retries: 0,
    batchMaxSize: 3,
    batchMaxInterval: '150ms',
    batchGroupKey: 'input.group',
    fn: (tasks) =>
      tasks.map(([input]) => ({
        batchKey: input.group,
        batchSize: tasks.length,
        uniqueKeys: new Set(tasks.map(([item]) => item.group)).size,
        payload: input.Message,
      })),
  });

  // Test 5: large payload
  const largePayloadWorkflow = hatchet.batchTask<
    { data: string },
    { received: boolean; batchSize: number; dataLength: number }
  >({
    name: `batch-e2e-large-${runId}`,
    retries: 0,
    batchMaxSize: 100,
    batchMaxInterval: '1000s',
    fn: (tasks) => {
      console.info('task length', tasks.length);
      return tasks.map(([input]) => ({
        received: true,
        batchSize: tasks.length,
        dataLength: input.data.length,
      }));
    },
  });

  // Test 6: batch size of one
  const singleItemWorkflow = hatchet.batchTask<
    { Message: string },
    { original: string; batchSize: number }
  >({
    name: `batch-e2e-single-${runId}`,
    retries: 0,
    batchMaxSize: 1,
    batchMaxInterval: '100ms',
    fn: (tasks) =>
      tasks.map(([input]) => ({
        original: input.Message,
        batchSize: tasks.length,
      })),
  });

  beforeAll(async () => {
    const allWorkflows = [
      batchWorkflow,
      keyedBatchWorkflow,
      keyedIntervalWorkflow,
      largePayloadWorkflow,
      singleItemWorkflow,
    ];

    worker = await hatchet.worker(`batch-e2e-worker-${runId}`, {
      workflows: allWorkflows,
      slots: 25,
    });

    void worker.start();
    await sleep(2000);
  });

  afterAll(async () => {
    await worker.stop();
    await sleep(2000);
  });

  it('flushes when batch size is reached', async () => {
    const inputs = ['alpha', 'bravo', 'charlie'];

    const results = await Promise.all(
      inputs.map((message) =>
        batchWorkflow.run({
          Message: message,
        })
      )
    );

    expect(results).toHaveLength(3);
    expect(results.map((result) => result.TransformedMessage)).toEqual(
      inputs.map((value) => value.toUpperCase())
    );
  });

  it('flushes when fewer items are buffered than the batch size', async () => {
    const inputs = ['delta', 'echo'];

    const promises = inputs.map((Message) =>
      batchWorkflow.run({
        Message,
      })
    );

    await sleep(500);

    const results = await Promise.all(promises);
    expect(results.map((result) => result.TransformedMessage)).toEqual(
      inputs.map((value) => value.toUpperCase())
    );
  });

  it('partitions batches by key when batch size is reached', async () => {
    const inputs: Array<{ Message: string; group: string }> = [
      { Message: 'alpha', group: 'tenant-1' },
      { Message: 'bravo', group: 'tenant-1' },
      { Message: 'charlie', group: 'tenant-2' },
      { Message: 'delta', group: 'tenant-2' },
    ];

    const results = await Promise.all(inputs.map((input) => keyedBatchWorkflow.run(input)));

    expect(results).toHaveLength(inputs.length);
    results.forEach((result, index) => {
      expect(result.batchKey).toBe(inputs[index].group);
      expect(result.batchSize).toBe(2);
      expect(result.uniqueKeys).toBe(1);
      expect(result.uppercase).toBe(inputs[index].Message.toUpperCase());
    });
  });

  it('flushes keyed batches independently when flush interval elapses', async () => {
    const inputs: Array<{ Message: string; group: string }> = [
      { Message: 'echo', group: 'tenant-1' },
      { Message: 'foxtrot', group: 'tenant-1' },
      { Message: 'golf', group: 'tenant-1' },
      { Message: 'hotel', group: 'tenant-2' },
    ];

    const results = await Promise.all(inputs.map((input) => keyedIntervalWorkflow.run(input)));

    expect(results.map((result) => result.batchKey)).toEqual(inputs.map((input) => input.group));
    expect(results.slice(0, 3).every((result) => result.batchSize === 3)).toBe(true);
    expect(results[3].batchSize).toBe(1);
    expect(results.every((result) => result.uniqueKeys === 1)).toBe(true);
    expect(results[3].payload).toBe('hotel');
  });

  it('completes all tasks when batch contains 100+ items with 100kb+ payloads', async () => {
    jest.setTimeout(120_000);

    const payload = 'x'.repeat(4_000_000); // ~400mb per task
    const taskCount = 100;

    const results = await Promise.all(
      Array.from({ length: taskCount }, () => largePayloadWorkflow.run({ data: payload }))
    );

    expect(results).toHaveLength(taskCount);
    expect(results.every((r) => r.received)).toBe(true);
    expect(results.every((r) => r.dataLength === 4_000_000)).toBe(true);
    expect(results.every((r) => r.batchSize === taskCount)).toBe(true);
  });

  it('handles batch size of one without keys', async () => {
    const inputs = ['india', 'juliet'];

    const results = await Promise.all(
      inputs.map((Message) =>
        singleItemWorkflow.run({
          Message,
        })
      )
    );

    expect(results.map((result) => result.batchSize)).toEqual([1, 1]);
    expect(results.map((result) => result.original)).toEqual(inputs);
  });
});
