import sleep from '@hatchet/util/sleep';
import type WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import { makeE2EClient, poll } from '../__e2e__/harness';
import { concurrencyCancelInProgressWorkflow } from './workflow';

describe('concurrency-cancel-in-progress-e2e', () => {
  const hatchet = makeE2EClient();

  it('cancels in-progress runs when newer run arrives', async () => {
    const testRunId = crypto.randomUUID();
    const refs: WorkflowRunRef<any>[] = [];

    for (let i = 0; i < 10; i += 1) {
      const ref = await concurrencyCancelInProgressWorkflow.runNoWait(
        { group: 'A' },
        { additionalMetadata: { test_run_id: testRunId, i: String(i) } }
      );
      refs.push(ref);
      await sleep(1000);
    }

    for (const ref of refs) {
      try {
        await ref.output;
      } catch {
        // Expected for cancelled runs
      }
    }

    const runsResp = await poll(
      async () =>
        hatchet.runs.list({
          additionalMetadata: { test_run_id: testRunId },
          onlyTasks: false,
        } as any),
      {
        timeoutMs: 30_000,
        intervalMs: 200,
        label: 'runs list with terminal statuses',
        shouldStop: (r) => {
          const rows = r.rows || [];
          return (
            rows.length === 10 &&
            rows.every(
              (x: any) =>
                x.status !== V1TaskStatus.RUNNING && x.status !== V1TaskStatus.QUEUED
            )
          );
        },
      }
    );
    const runs = (runsResp.rows || []).sort(
      (a: any, b: any) =>
        parseInt(String((a.additionalMetadata as Record<string, unknown>)?.i ?? '0'), 10) -
        parseInt(String((b.additionalMetadata as Record<string, unknown>)?.i ?? '0'), 10)
    );

    expect(runs).toHaveLength(10);
    expect((runs[9].additionalMetadata as Record<string, unknown>)?.i).toBe('9');
    expect(runs[9].status).toBe(V1TaskStatus.COMPLETED);
    expect(runs.slice(0, 9).every((r: any) => r.status === V1TaskStatus.CANCELLED)).toBe(true);
  }, 120_000);
});
