import { makeE2EClient, poll } from '../__e2e__/harness';
import { helloWorld, helloWorldDurable } from './e2e-workflows';
import { V1TaskStatus } from '../../../clients/rest/generated/data-contracts';

describe('simple-run-modes-e2e', () => {
  const hatchet = makeE2EClient();

  it('supports the run variants for tasks and durable tasks', async () => {
    const expected = { result: 'Hello, world!' };

    for (const task of [helloWorld, helloWorldDurable]) {
      const x1 = await task.run({});
      const x2 = await (await task.runNoWait({})).output;

      const [x3] = await task.run([{}]);
      const x4 = await (await task.runNoWait([{}]))[0].output;

      // output alias for output
      const x5 = await (await task.runNoWait({})).output;

      expect([x1, x2, x3, x4, x5]).toEqual([expected, expected, expected, expected, expected]);
    }
  }, 90_000);

  it('supports runMany variants with per-item options', async () => {
    const expected = { result: 'Hello, world!' };

    for (const task of [helloWorld, helloWorldDurable]) {
      const runs = [
        { input: {}, opts: { additionalMetadata: { test_case: 'run-many-a' } } },
        {
          input: {},
          opts: { additionalMetadata: { test_case: 'run-many-b' }, priority: 3 },
        },
      ];

      const waited = await task.runMany(runs);
      expect(waited).toEqual([expected, expected]);

      const refs = await task.runManyNoWait(runs);
      const outputs = await Promise.all(refs.map((r) => r.output));
      expect(outputs).toEqual([expected, expected]);

      const [detailsA, detailsB] = await Promise.all(
        refs.map((ref) =>
          poll(async () => hatchet.runs.get(ref), {
            timeoutMs: 30_000,
            intervalMs: 100,
            label: 'run details available',
            shouldStop: (d) =>
              [V1TaskStatus.QUEUED, V1TaskStatus.RUNNING, V1TaskStatus.COMPLETED].includes(
                d.run.status as any
              ),
          })
        )
      );

      expect((detailsA.run as any).additionalMetadata || {}).toMatchObject({
        test_case: 'run-many-a',
      });
      expect((detailsB.run as any).additionalMetadata || {}).toMatchObject({
        test_case: 'run-many-b',
      });
    }
  }, 120_000);
});
