import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';
import { returnExceptionsTask } from './workflow';

describe('return-exceptions-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'return-exceptions-e2e-worker',
      workflows: [returnExceptionsTask],
      slots: 10,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it('run with returnExceptions returns mixed successes and errors', async () => {
    const results = await returnExceptionsTask.run(
      Array.from({ length: 10 }, (_, i) => ({ index: i })),
      { returnExceptions: true }
    );

    expect(results).toHaveLength(10);

    for (let i = 0; i < 10; i += 1) {
      if (i % 2 === 0) {
        expect(results[i]).toBeInstanceOf(Error);
        expect((results[i] as Error).message).toContain(`error in task with index ${i}`);
      } else {
        expect(results[i]).toEqual({ message: 'this is a successful task.' });
      }
    }
  }, 60_000);
});
