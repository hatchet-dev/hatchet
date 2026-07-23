import sleep from '@hatchet/util/sleep';
import {
  batchSimple,
  batchKeyed,
  batchKeyedFailable,
  batchKeyedInterval,
  batchLarge,
  batchSingle,
  batchOrdered,
  batchBroadcast,
  batchCancel,
  batchChildSpawn,
  batchChildBatchSpawn,
} from './workflow';

describe('batch-assign-e2e', () => {
  it('flushes when batch size is reached', async () => {
    const inputs = ['alpha', 'bravo', 'charlie'];

    const results = await Promise.all(inputs.map((message) => batchSimple.run({ message })));

    expect(results.map((r) => r.transformed_message)).toEqual(inputs.map((m) => m.toUpperCase()));
  }, 30_000);

  it('flushes on interval when fewer items than batch size are buffered', async () => {
    const inputs = ['delta', 'echo'];

    const refs = await Promise.all(inputs.map((message) => batchSimple.runNoWait({ message })));
    await sleep(500);

    const results = await Promise.all(refs.map((ref) => ref.output));

    expect(results.map((r: any) => r.transformed_message)).toEqual(
      inputs.map((m) => m.toUpperCase())
    );
  }, 30_000);

  it('partitions batches by key when batch size is reached', async () => {
    const inputs = [
      { message: 'alpha', group: 'tenant-1' },
      { message: 'bravo', group: 'tenant-1' },
      { message: 'charlie', group: 'tenant-2' },
      { message: 'delta', group: 'tenant-2' },
    ];

    const results = await Promise.all(inputs.map((input) => batchKeyed.run(input)));

    inputs.forEach((input, i) => {
      expect(results[i].batch_key).toEqual(input.group);
      expect(results[i].batch_size).toEqual(2);
      expect(results[i].unique_keys).toEqual(1);
      expect(results[i].uppercase).toEqual(input.message.toUpperCase());
    });
  }, 30_000);

  it('fails only the task whose batch group key fails to parse', async () => {
    const goodRef = await batchKeyedFailable.runNoWait({ message: 'hello', group: 'tenant-1' });
    const badRef = await batchKeyedFailable.runNoWait({ message: 'world', group: 123 });

    await expect(badRef.output).rejects.toEqual(
      expect.arrayContaining([
        expect.stringContaining('failed to parse batch group key expression'),
      ])
    );

    const goodResult: any = await goodRef.output;
    expect(goodResult.uppercase).toEqual('HELLO');
  }, 30_000);

  it('flushes keyed batches independently when interval elapses', async () => {
    const inputs = [
      { message: 'echo', group: 'tenant-1' },
      { message: 'foxtrot', group: 'tenant-1' },
      { message: 'golf', group: 'tenant-1' },
      { message: 'hotel', group: 'tenant-2' },
    ];

    const results = await Promise.all(inputs.map((input) => batchKeyedInterval.run(input)));

    inputs.forEach((input, i) => expect(results[i].batch_key).toEqual(input.group));
    [0, 1, 2].forEach((i) => expect(results[i].batch_size).toEqual(3));
    expect(results[3].batch_size).toEqual(1);
    results.forEach((r) => expect(r.unique_keys).toEqual(1));
    expect(results[3].uppercase).toEqual('HOTEL');
  }, 30_000);

  it('completes all tasks with large payloads, flushing on memory size', async () => {
    const payloadSize = 100_000;
    const payload = 'x'.repeat(payloadSize);
    const taskCount = 100;

    const results = await Promise.all(
      Array.from({ length: taskCount }, () => batchLarge.run({ data: payload }))
    );

    expect(results.length).toEqual(taskCount);
    // The batch should flush 3 times due to the ~4mb memory-size limit, even though
    // batch_max_size (100) is never reached by count alone.
    expect(new Set(results.map((r) => r.batch_id)).size).toEqual(3);
    results.forEach((r) => {
      expect(r.received).toBe(true);
      expect(r.data_length).toEqual(payloadSize);
    });
  }, 60_000);

  it('handles batch size of one without keys', async () => {
    const inputs = ['india', 'juliet'];

    const results = await Promise.all(inputs.map((message) => batchSingle.run({ message })));

    expect(results.map((r) => r.batch_size)).toEqual([1, 1]);
    expect(results.map((r) => r.original)).toEqual(inputs);
  }, 30_000);

  it('returns results in submission order', async () => {
    const count = 20;

    const results = await Promise.all(
      Array.from({ length: count }, (_, index) => batchOrdered.run({ index }))
    );

    results.forEach((result, i) => expect(result.index).toEqual(i));
  }, 30_000);

  it('broadcasts the same result to all callers', async () => {
    const count = 10;

    const results = await Promise.all(
      Array.from({ length: count }, () => batchBroadcast.run({ message: 'hello' }))
    );

    results.forEach((r) => expect(r.sum).toEqual(50));
  }, 30_000);

  it('supports in-batch cancellation via ctx.cancel', async () => {
    const count = 10;

    const results = await Promise.all(
      Array.from({ length: count }, () => batchCancel.run({ message: 'hello' }))
    );

    expect(results.length).toEqual(count);
  }, 30_000);

  it('supports spawning child tasks from a batch handler', async () => {
    const count = 10;

    const results = await Promise.all(
      Array.from({ length: count }, () => batchChildSpawn.run({ message: 'hello' }))
    );

    results.forEach((r) => expect(Object.keys(r).length).toBeGreaterThan(0));
  }, 30_000);

  it('supports spawning child batch tasks from a batch handler', async () => {
    const count = 10;

    const results = await Promise.all(
      Array.from({ length: count }, () => batchChildBatchSpawn.run({ message: 'hello' }))
    );

    results.forEach((r) => expect(Object.keys(r).length).toBeGreaterThan(0));
  }, 30_000);
});
