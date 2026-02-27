import sleep from '@hatchet/util/sleep';
import { HatchetClient } from '@hatchet/v1';
import { Worker } from '../../client/worker/worker';

describe('middleware-e2e', () => {
  let worker: Worker;

  afterEach(async () => {
    if (worker) {
      await worker.stop();
      await sleep(2000);
    }
  });

  it('should inject before middleware fields into task input and after middleware fields into output', async () => {
    const client = HatchetClient.init<
      { first: number; second: number },
      { extra: number }
    >().withMiddleware({
      before: (input) => {
        return { ...input, dependency: `dep-${input.first}-${input.second}` };
      },
      after: (output) => {
        return { ...output, additionalData: 42 };
      },
    });

    const task = client.task<{ message: string }, { message: string; extra: number }>({
      name: 'middleware-e2e-single',
      fn: (input) => {
        return {
          message: `${input.message}:${input.dependency}`,
          extra: input.first + input.second,
        };
      },
    });

    worker = await client.worker('middleware-e2e-worker', {
      workflows: [task],
    });

    void worker.start();
    await sleep(5000);

    const result = await task.run({
      message: 'hello',
      first: 10,
      second: 20,
    });

    expect(result.message).toBe('hello:dep-10-20');
    expect(result.extra).toBe(30);
    expect(result.additionalData).toBe(42);
  }, 60000);

  it('should strip fields not included in middleware return when input is not spread', async () => {
    const client = HatchetClient.init<{ first: number; second: number }>().withMiddleware({
      before: (input) => {
        return { dependency: `dep-${input.first}-${input.second}` };
      },
      after: (output) => {
        return { additionalData: 99 };
      },
    });

    const task = client.task<{}, { result: string }>({
      name: 'middleware-e2e-no-spread',
      fn: (input) => {
        return {
          result: input.dependency,
        };
      },
    });

    worker = await client.worker('middleware-e2e-no-spread-worker', {
      workflows: [task],
    });

    void worker.start();
    await sleep(5000);

    const result = await task.run({ first: 10, second: 20 });

    expect(result.additionalData).toBe(99);
    expect((result as any).result).toBeUndefined();
  }, 60000);

  it('should chain multiple withMiddleware calls with accumulated context', async () => {
    const client = HatchetClient.init<{ value: number }>()
      .withMiddleware({
        before: (input) => {
          return { ...input, doubled: input.value * 2 };
        },
        after: (output) => {
          return { ...output, postFirst: true };
        },
      })
      .withMiddleware({
        before: (input) => {
          return { ...input, quadrupled: input.doubled * 2 };
        },
        after: (output) => {
          return { ...output, postSecond: true };
        },
      });

    const task = client.task<{}, { result: number }>({
      name: 'middleware-e2e-chained',
      fn: (input) => {
        return {
          result: input.quadrupled,
        };
      },
    });

    worker = await client.worker('middleware-e2e-chained-worker', {
      workflows: [task],
    });

    void worker.start();
    await sleep(5000);

    const result = await task.run({ value: 5 });

    expect(result.result).toBe(20);
    expect(result.postFirst).toBe(true);
    expect(result.postSecond).toBe(true);
  }, 60000);
});
