import { makeE2EClient } from '../__e2e__/harness';
import { helloWorld, helloWorldDurable } from './e2e-workflows';

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
});
