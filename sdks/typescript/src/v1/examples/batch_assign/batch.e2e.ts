import { randomUUID } from 'crypto';
import sleep from '@hatchet/util/sleep';
import { hatchet } from '../hatchet-client';
import { Worker } from '../../client/worker/worker';
import type { JsonObject } from '../../types';

describe('batch-task e2e', () => {
  jest.setTimeout(60000);

  let worker: Worker;
  const workflowName = `batch-e2e-${randomUUID()}`;

  const batchWorkflow = hatchet.batchTask({
    name: workflowName,
    retries: 0,
    batchSize: 3,
    flushInterval: 200,
    fn: (inputs: { Message: string }[]) =>
      inputs.map((input) => ({
        TransformedMessage: input.Message.toUpperCase(),
      })),
  });

  const createAndRegisterBatchWorkflow = async <
    I extends JsonObject,
    O extends JsonObject,
  >(config: {
    fn: (inputs: I[]) => O[] | Promise<O[]>;
    batchSize: number;
    flushInterval?: number;
    batchKey?: string;
    maxRuns?: number;
    name?: string;
    retries?: number;
  }) => {
    if (!worker) {
      throw new Error('Worker not initialized');
    }

    const { fn, batchSize, flushInterval, batchKey, maxRuns, name, retries } = config;

    const workflow = hatchet.batchTask<I, O>({
      name: name ?? `batch-e2e-${randomUUID()}`,
      retries: retries ?? 0,
      batchSize,
      flushInterval,
      batchKey,
      maxRuns,
      fn,
    });

    await worker.registerWorkflow(workflow);
    await sleep(200);

    return workflow;
  };

  beforeAll(async () => {
    worker = await hatchet.worker(`batch-e2e-worker-${randomUUID()}`, {
      workflows: [batchWorkflow],
      slots: 25,
    });

    await worker.registerWorkflow(batchWorkflow);
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
    const keyedWorkflow = await createAndRegisterBatchWorkflow({
      batchSize: 2,
      flushInterval: 200,
      batchKey: 'input.group',
      fn: (inputs: Array<{ Message: string; group: string }>) =>
        inputs.map((input) => ({
          batchKey: input.group,
          batchSize: inputs.length,
          uniqueKeys: new Set(inputs.map((item) => item.group)).size,
          uppercase: input.Message.toUpperCase(),
        })),
    });

    const inputs: Array<{ Message: string; group: string }> = [
      { Message: 'alpha', group: 'tenant-1' },
      { Message: 'bravo', group: 'tenant-1' },
      { Message: 'charlie', group: 'tenant-2' },
      { Message: 'delta', group: 'tenant-2' },
    ];

    const results = await Promise.all(inputs.map((input) => keyedWorkflow.run(input)));

    expect(results).toHaveLength(inputs.length);
    results.forEach((result, index) => {
      expect(result.batchKey).toBe(inputs[index].group);
      expect(result.batchSize).toBe(2);
      expect(result.uniqueKeys).toBe(1);
      expect(result.uppercase).toBe(inputs[index].Message.toUpperCase());
    });
  });

  it('flushes keyed batches independently when flush interval elapses', async () => {
    const keyedWorkflow = await createAndRegisterBatchWorkflow({
      batchSize: 3,
      flushInterval: 150,
      batchKey: 'input.group',
      fn: (inputs: Array<{ Message: string; group: string }>) =>
        inputs.map((input) => ({
          batchKey: input.group,
          batchSize: inputs.length,
          uniqueKeys: new Set(inputs.map((item) => item.group)).size,
          payload: input.Message,
        })),
    });

    const inputs: Array<{ Message: string; group: string }> = [
      { Message: 'echo', group: 'tenant-1' },
      { Message: 'foxtrot', group: 'tenant-1' },
      { Message: 'golf', group: 'tenant-1' },
      { Message: 'hotel', group: 'tenant-2' },
    ];

    const results = await Promise.all(inputs.map((input) => keyedWorkflow.run(input)));

    expect(results.map((result) => result.batchKey)).toEqual(
      inputs.map((input) => input.group)
    );
    expect(results.slice(0, 3).every((result) => result.batchSize === 3)).toBe(true);
    expect(results[3].batchSize).toBe(1);
    expect(results.every((result) => result.uniqueKeys === 1)).toBe(true);
    expect(results[3].payload).toBe('hotel');
  });

  it('handles batch size of one without keys', async () => {
    const singleItemWorkflow = await createAndRegisterBatchWorkflow({
      batchSize: 1,
      flushInterval: 100,
      fn: (inputs: Array<{ Message: string }>) =>
        inputs.map((input) => ({
          original: input.Message,
          batchSize: inputs.length,
        })),
    });

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
