import { WorkflowDeclaration } from '../../declaration';
import { mapBatchConfigPb } from './worker-internal';

// Never called. It exists so tsc checks that durable task options reject batch; if the
// omission regresses, the @ts-expect-error goes unused and tsc fails.
export function durableTaskRejectsBatch() {
  const wf = new WorkflowDeclaration({ name: 'batch-type-check' });
  wf.durableTask({
    name: 'd',
    // @ts-expect-error batch is not available on durable tasks
    batch: { maxSize: 3 },
    fn: async () => undefined,
  });
}

describe('mapBatchConfigPb', () => {
  it('returns undefined when batch is not set', () => {
    expect(mapBatchConfigPb(undefined)).toBeUndefined();
  });

  it('maps required and optional fields to the proto shape', () => {
    expect(
      mapBatchConfigPb({
        maxSize: 5,
        maxInterval: 200,
        groupKey: 'input.group',
        groupMaxRuns: 3,
        broadcastOutput: true,
      })
    ).toEqual({
      batchMaxSize: 5,
      batchMaxIntervalMs: 200,
      batchGroupKey: 'input.group',
      batchGroupMaxRuns: 3,
      broadcastOutput: true,
    });
  });

  it('converts a numeric maxInterval (milliseconds) directly', () => {
    expect(mapBatchConfigPb({ maxSize: 3, maxInterval: 200 })?.batchMaxIntervalMs).toEqual(200);
  });

  it('converts a duration-object maxInterval to milliseconds', () => {
    expect(
      mapBatchConfigPb({ maxSize: 3, maxInterval: { seconds: 2 } })?.batchMaxIntervalMs
    ).toEqual(2000);
  });

  it('leaves optional fields undefined when omitted', () => {
    expect(mapBatchConfigPb({ maxSize: 1 })).toEqual({
      batchMaxSize: 1,
      batchMaxIntervalMs: undefined,
      batchGroupKey: undefined,
      batchGroupMaxRuns: undefined,
      broadcastOutput: undefined,
    });
  });

  it('rejects a maxSize of 0 or a negative maxSize', () => {
    expect(() => mapBatchConfigPb({ maxSize: 0 })).toThrow(/positive integer/);
    expect(() => mapBatchConfigPb({ maxSize: -1 })).toThrow(/positive integer/);
  });

  it('rejects a non-integer maxSize', () => {
    expect(() => mapBatchConfigPb({ maxSize: 2.5 })).toThrow(/positive integer/);
  });

  it('rejects a maxInterval that rounds down to zero or negative milliseconds', () => {
    expect(() => mapBatchConfigPb({ maxSize: 1, maxInterval: 0 })).toThrow(/must be positive/);
  });

  it('rejects a groupMaxRuns of 0 or a negative groupMaxRuns', () => {
    expect(() => mapBatchConfigPb({ maxSize: 1, groupMaxRuns: 0 })).toThrow(/positive integer/);
    expect(() => mapBatchConfigPb({ maxSize: 1, groupMaxRuns: -1 })).toThrow(/positive integer/);
  });
});

describe('hatchet.batchTask / workflow.batchTask registration', () => {
  it('forces retries to 0 on the task definition even when explicitly set', () => {
    const wf = new WorkflowDeclaration({ name: 'batch-retries-workflow' });
    const batchTask = wf.batchTask({
      name: 'batch-task',
      batch: { maxSize: 3 },
      retries: 5,
      fn: async (inputs: Record<string, any>) =>
        Object.fromEntries(Object.keys(inputs).map((id) => [id, {}])),
    });

    expect(batchTask.batch).toEqual({ maxSize: 3 });
  });

  it('broadcasts a single handler result to every batch member id', async () => {
    const wf = new WorkflowDeclaration({ name: 'batch-broadcast-workflow' });
    const batchTask = wf.batchTask({
      name: 'batch-broadcast-task',
      batch: { maxSize: 3, broadcastOutput: true },
      fn: async () => ({ sum: 7 }),
    });

    const fn = batchTask.fn!;
    const result = await fn({ 'id-1': { message: 'a' }, 'id-2': { message: 'b' } }, {} as any);

    expect(result).toEqual({ 'id-1': { sum: 7 }, 'id-2': { sum: 7 } });
  });

  it('passes through a per-member result Record unchanged when not broadcasting', async () => {
    const wf = new WorkflowDeclaration({ name: 'batch-dict-workflow' });
    const batchTask = wf.batchTask({
      name: 'batch-dict-task',
      batch: { maxSize: 3 },
      fn: async (inputs: Record<string, any>) =>
        Object.fromEntries(
          Object.entries(inputs).map(([id, input]) => [id, { upper: input.message.toUpperCase() }])
        ),
    });

    const fn = batchTask.fn!;
    const result = await fn(
      { 'id-1': { message: 'hello' }, 'id-2': { message: 'world' } },
      {} as any
    );

    expect(result).toEqual({ 'id-1': { upper: 'HELLO' }, 'id-2': { upper: 'WORLD' } });
  });

  it('throws when the non-broadcast handler result key set does not match the input', async () => {
    const wf = new WorkflowDeclaration({ name: 'batch-mismatch-workflow' });
    const batchTask = wf.batchTask({
      name: 'batch-mismatch-task',
      batch: { maxSize: 3 },
      fn: async () => ({ 'id-1': {} }),
    });

    const fn = batchTask.fn!;

    await expect(fn({ 'id-1': {}, 'id-2': {} }, {} as any)).rejects.toThrow(
      /do not match batch member ids/
    );
  });
});
