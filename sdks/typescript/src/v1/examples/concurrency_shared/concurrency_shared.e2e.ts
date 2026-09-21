import { randomUUID } from 'crypto';
import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';
import {
  concurrencySharedMixedWorkflow,
  concurrencySharedWorkflowA,
  concurrencySharedWorkflowB,
  RunWindow,
  WorkflowInput,
  WorkflowOutput,
} from './workflow';

const TOLERANCE_MS = 100;

type SharedWorkflow = typeof concurrencySharedWorkflowA;

async function gatherRunWindows(
  runs: Array<[SharedWorkflow, WorkflowInput]>
): Promise<RunWindow[]> {
  const results = await Promise.all(runs.map(([wf, input]) => wf.run(input)));
  return results.map((r) => (r as WorkflowOutput)['shared-task']);
}

/** No two run windows may overlap (max concurrency of 1). */
function expectSerialized(windows: RunWindow[]): void {
  const ordered = [...windows].sort((a, b) => a.startMs - b.startMs);

  for (let i = 1; i < ordered.length; i += 1) {
    const prev = ordered[i - 1];
    const cur = ordered[i];
    if (cur.startMs < prev.endMs - TOLERANCE_MS) {
      throw new Error(`runs overlapped: ${JSON.stringify(prev)} vs ${JSON.stringify(cur)}`);
    }
  }
}

/** At least one pair of windows must overlap, proving the worker can run these tasks
 * concurrently when no limit binds. */
function expectSomeOverlap(windows: RunWindow[]): void {
  for (let i = 0; i < windows.length; i += 1) {
    for (let j = i + 1; j < windows.length; j += 1) {
      if (windows[i].startMs < windows[j].endMs && windows[j].startMs < windows[i].endMs) {
        return;
      }
    }
  }

  throw new Error(`expected at least one overlapping pair, got: ${JSON.stringify(windows)}`);
}

describe('concurrency-shared-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'concurrency-shared-e2e-worker',
      workflows: [
        concurrencySharedWorkflowA,
        concurrencySharedWorkflowB,
        concurrencySharedMixedWorkflow,
      ],
      slots: 20,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it('tasks from different workflows sharing a strategy never run concurrently', async () => {
    const group = `xwf-${randomUUID()}`;

    const windows = await gatherRunWindows([
      [concurrencySharedWorkflowA, { group }],
      [concurrencySharedWorkflowA, { group }],
      [concurrencySharedWorkflowB, { group }],
      [concurrencySharedWorkflowB, { group }],
    ]);

    expectSerialized(windows);
  }, 120_000);

  it('mixed task: the shared limit binds across workflows when inline keys differ', async () => {
    const group = `mixed-shared-${randomUUID()}`;

    const windows = await gatherRunWindows([
      [concurrencySharedMixedWorkflow, { group, inline: `inline-${randomUUID()}` }],
      [concurrencySharedMixedWorkflow, { group, inline: `inline-${randomUUID()}` }],
      [concurrencySharedWorkflowA, { group }],
    ]);

    expectSerialized(windows);
  }, 120_000);

  it('mixed task: the inline limit binds when shared group keys differ', async () => {
    const inline = `inline-${randomUUID()}`;

    const windows = await gatherRunWindows([
      [concurrencySharedMixedWorkflow, { group: `group-${randomUUID()}`, inline }],
      [concurrencySharedMixedWorkflow, { group: `group-${randomUUID()}`, inline }],
    ]);

    expectSerialized(windows);
  }, 120_000);

  it('control: with no colliding keys, runs overlap freely', async () => {
    const windows = await gatherRunWindows([
      [
        concurrencySharedMixedWorkflow,
        { group: `group-${randomUUID()}`, inline: `inline-${randomUUID()}` },
      ],
      [
        concurrencySharedMixedWorkflow,
        { group: `group-${randomUUID()}`, inline: `inline-${randomUUID()}` },
      ],
    ]);

    expectSomeOverlap(windows);
  }, 120_000);
});
