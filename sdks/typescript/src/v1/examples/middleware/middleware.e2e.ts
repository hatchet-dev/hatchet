import sleep from '@hatchet/util/sleep';
import { HatchetClient } from '@hatchet/v1';
import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';

describe('middleware-e2e', () => {
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(() => {
    makeE2EClient(); // validate env before any test
  });

  afterEach(async () => {
    await stopWorker(worker);
    worker = undefined;
  });

  it('should inject before middleware fields into task input and after middleware fields into output', async () => {
    const client = HatchetClient.init<
      { first: number; second: number },
      { extra: number }
    >().withMiddleware({
      before: (input) => ({ ...input, dependency: `dep-${input.first}-${input.second}` }),
      after: (output) => ({ ...output, additionalData: 42 }),
    });

    const task = client.task<{ message: string }, { message: string; extra: number }>({
      name: 'middleware-e2e-single',
      fn: (input) => ({
          message: `${input.message}:${input.dependency}`,
          extra: input.first + input.second,
        }),
    });

    worker = await startWorker({
      client,
      name: 'middleware-e2e-worker',
      workflows: [task],
    });
    await sleep(500); // allow worker to fully subscribe before sending work

    const result = await task.run({
      message: 'hello',
      first: 10,
      second: 20,
    });

    expect(result.message).toBe('hello:dep-10-20');
    expect(result.extra).toBe(30);
    expect(result.additionalData).toBe(42);
  }, 90_000);

  it('should strip fields not included in middleware return when input is not spread', async () => {
    const client = HatchetClient.init<{ first: number; second: number }>().withMiddleware({
      before: (input) => ({ dependency: `dep-${input.first}-${input.second}` }),
      after: (output) => ({ additionalData: 99 }),
    });

    const task = client.task<{}, { result: string }>({
      name: 'middleware-e2e-no-spread',
      fn: (input) => ({
          result: input.dependency,
        }),
    });

    worker = await startWorker({
      client,
      name: 'middleware-e2e-no-spread-worker',
      workflows: [task],
    });
    await sleep(500);

    const result = await task.run({ first: 10, second: 20 });

    expect(result.additionalData).toBe(99);
    expect((result as any).result).toBeUndefined();
  }, 90_000);

  it('should chain multiple withMiddleware calls with accumulated context', async () => {
    const client = HatchetClient.init<{ value: number }>()
      .withMiddleware({
        before: (input) => ({ ...input, doubled: input.value * 2 }),
        after: (output) => ({ ...output, postFirst: true }),
      })
      .withMiddleware({
        before: (input) => ({ ...input, quadrupled: input.doubled * 2 }),
        after: (output) => ({ ...output, postSecond: true }),
      });

    const task = client.task<{}, { result: number }>({
      name: 'middleware-e2e-chained',
      fn: (input) => ({
          result: input.quadrupled,
        }),
    });

    worker = await startWorker({
      client,
      name: 'middleware-e2e-chained-worker',
      workflows: [task],
    });
    await sleep(500);

    const result = await task.run({ value: 5 });

    expect(result.result).toBe(20);
    expect(result.postFirst).toBe(true);
    expect(result.postSecond).toBe(true);
  }, 90_000);
});
