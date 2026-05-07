import { applyNamespace } from '@hatchet/util/apply-namespace';
import { makeE2EClient, poll, makeTestScope } from '../__e2e__/harness';
import { V1TaskStatus } from '../../../clients/rest/generated/data-contracts';
import { bulkReplayTest1, bulkReplayTest2, bulkReplayTest3 } from './workflow';

describe('bulk-replay-e2e', () => {
  const hatchet = makeE2EClient();

  it('bulk replays matching runs and increments retry count', async () => {
    const testRunId = makeTestScope('bulk_replay');
    const n = 20;

    const meta = { test_run_id: testRunId };
    const since = new Date(Date.now() - 5 * 60 * 1000);

    const inputs = (count: number) => Array.from({ length: count }, () => ({}));

    await bulkReplayTest1.runNoWait(inputs(n + 1), { additionalMetadata: meta });
    await bulkReplayTest2.runNoWait(inputs(n / 2 - 1), { additionalMetadata: meta });
    await bulkReplayTest3.runNoWait(inputs(n / 2 - 2), { additionalMetadata: meta });

    const workflowNames = [bulkReplayTest1.name, bulkReplayTest2.name, bulkReplayTest3.name];
    const expectedTotal = n + 1 + (n / 2 - 1) + (n / 2 - 2);

    const initialRuns = await poll(
      async () =>
        hatchet.runs.list({
          since,
          limit: 1000,
          workflowNames,
          additionalMetadata: meta,
          onlyTasks: true,
        }),
      {
        timeoutMs: 120_000,
        intervalMs: 200,
        label: 'initial bulk runs completion',
        shouldStop: (runs) =>
          (runs.rows || []).length === expectedTotal &&
          (runs.rows || []).every((r: any) => r.status === V1TaskStatus.COMPLETED),
      }
    );

    expect(initialRuns.rows).toHaveLength(expectedTotal);
    const initialRetryCounts = new Map(
      (initialRuns.rows || []).map((r: any) => [r.metadata.id, r.retryCount ?? 0])
    );

    // Equivalent to Python's aio_bulk_replay: runs.replay with filter.
    await hatchet.runs.replay({
      filters: {
        since,
        workflowNames,
        additionalMetadata: meta,
      },
    });

    const replayedRuns = await poll(
      async () =>
        hatchet.runs.list({
          since,
          limit: 1000,
          workflowNames,
          additionalMetadata: meta,
          onlyTasks: true,
        }),
      {
        timeoutMs: 120_000,
        intervalMs: 200,
        label: 'bulk replay retry counts visible',
        shouldStop: (runs) =>
          (runs.rows || []).length === expectedTotal &&
          (runs.rows || []).every(
            (r: any) =>
              r.status === V1TaskStatus.COMPLETED &&
              (r.retryCount ?? 0) >
                (initialRetryCounts.get(r.metadata.id) ?? Number.MAX_SAFE_INTEGER)
          ),
      }
    );

    const rows = replayedRuns.rows || [];
    expect(rows).toHaveLength(expectedTotal);

    const { namespace } = hatchet.config;
    const byName = (name: string) => {
      const expectedName = applyNamespace(name, namespace);
      return rows.filter((r: any) => r.workflowName === expectedName);
    };
    expect(byName(bulkReplayTest1.name)).toHaveLength(n + 1);
    expect(byName(bulkReplayTest2.name)).toHaveLength(n / 2 - 1);
    expect(byName(bulkReplayTest3.name)).toHaveLength(n / 2 - 2);
  }, 240_000);
});
