import { makeE2EClient, poll } from '../__e2e__/harness';
import { cancellationWorkflow } from './cancellation-workflow';
import { V1TaskStatus } from '../../../clients/rest/generated/data-contracts';

describe('cancellation-e2e', () => {
  const hatchet = makeE2EClient();

  xit('should cancel eventually (execution timeout)', async () => {
    const ref = await cancellationWorkflow.runNoWait({});

    const run = await poll(async () => hatchet.runs.get(ref), {
      timeoutMs: 60_000,
      intervalMs: 100,
      label: 'cancellation run status',
      shouldStop: (r) => ![V1TaskStatus.QUEUED, V1TaskStatus.RUNNING].includes(r.run.status as any),
    });

    expect(run.run.status).toBe(V1TaskStatus.CANCELLED);

    // best-effort: python asserts `not run.run.output`
    const out: unknown = run.run.output;
    const isEmptyObject =
      out != null &&
      typeof out === 'object' &&
      Object.keys(out as Record<string, unknown>).length === 0;
    expect(out == null || isEmptyObject).toBeTruthy();
  }, 90_000);
});
