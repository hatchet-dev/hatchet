import { randomUUID } from 'crypto';
import { makeE2EClient, poll } from '../__e2e__/harness';
import { bulkChild, bulkParentWorkflow } from './workflow';

describe('bulk-fanout-e2e', () => {
  const hatchet = makeE2EClient();

  it('spawns N children and returns all results', async () => {
    const result = await bulkParentWorkflow.run({ n: 12 });
    expect((result as any).spawn.results).toHaveLength(12);
  }, 90_000);

  it('propagates additionalMetadata to parent and child runs', async () => {
    const testRunId = randomUUID();

    const ref = await bulkParentWorkflow.runNoWait(
      { n: 2 },
      { additionalMetadata: { test_run_id: testRunId } }
    );

    await ref.output;
    const details = await poll(
      async () => hatchet.runs.get(ref),
      {
        timeoutMs: 15_000,
        intervalMs: 100,
        label: 'run details with metadata',
        shouldStop: (d) =>
          (d.run?.additionalMetadata as Record<string, string>)?.test_run_id === testRunId,
      }
    );
    expect((details.run?.additionalMetadata as Record<string, string>)?.test_run_id).toBe(
      testRunId
    );

    const spawnTask = (details.tasks || []).find((t) => (t.numSpawnedChildren ?? 0) > 0);
    const parentTaskId = spawnTask?.taskExternalId;
    if (parentTaskId) {
      const runsResp = await hatchet.runs.list({
        parentTaskRunExternalId: parentTaskId,
        onlyTasks: false,
      });

      if ((runsResp.rows?.length ?? 0) >= 2) {
        for (const run of runsResp.rows || []) {
          const meta = (run.additionalMetadata as Record<string, string>) || {};
          expect(meta.test_run_id).toBe(testRunId);
        }
      }
    }

    const runsByMeta = await hatchet.runs.list({
      additionalMetadata: { test_run_id: testRunId },
      onlyTasks: false,
    });
    expect(runsByMeta.rows && runsByMeta.rows.length).toBeGreaterThan(0);
  }, 90_000);
});
