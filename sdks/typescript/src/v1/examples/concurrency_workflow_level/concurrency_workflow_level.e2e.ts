import { V1TaskSummary } from '@hatchet/clients/rest/generated/data-contracts';
import { makeE2EClient, poll } from '../__e2e__/harness';
import { concurrencyWorkflowLevelWorkflow, DIGIT_MAX_RUNS, NAME_MAX_RUNS } from './workflow';

const CHARACTERS = ['Anna', 'Vronsky', 'Stiva', 'Dolly', 'Levin', 'Karenin'] as const;
const NUM_RUNS = 100;
const TASKS_PER_RUN = 2;

function pick<T>(arr: readonly T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

type RunWindow = {
  runId: string;
  name: string;
  digit: string;
  startedAt: number;
  finishedAt: number;
};

// A workflow run holds its workflow-level concurrency slots from the moment its
// first task starts until its last task finishes, so its execution window is the
// union of its task windows.
//
// The windows are built from task rows (onlyTasks: true) on purpose: task
// startedAt/finishedAt come from the STARTED/FINISHED event timestamps stamped
// by the worker, so a run that only starts after another run released a slot
// always has a later startedAt than that run's finishedAt. DAG rows
// (onlyTasks: false) instead derive these timestamps from the OLAP write time
// of the events, which lags and reorders under load and makes sequential
// handoffs look like overlaps.
// Returns the instant as integer microseconds since the epoch. Date only keeps
// milliseconds, which would let a start and a finish that are microseconds
// apart fall into the same instant and be read as a handoff; the API returns
// the full fractional seconds, so keep them.
function toMicros(timestamp: string): number {
  const match = /^(.*T[^.Z+-]*)(?:\.(\d+))?(Z|[+-]\d\d:\d\d)$/.exec(timestamp);
  if (!match) {
    throw new Error(`unexpected timestamp format: ${timestamp}`);
  }

  const [, base, fraction = '', zone] = match;
  const wholeSeconds = Math.floor(Date.parse(`${base}${zone}`) / 1000);
  const micros = Number(`${fraction}000000`.slice(0, 6));

  return wholeSeconds * 1_000_000 + micros;
}

function buildRunWindows(tasks: V1TaskSummary[]): RunWindow[] {
  const windows: Record<string, RunWindow> = {};

  for (const task of tasks) {
    if (!task.startedAt || !task.finishedAt) {
      throw new Error(`task ${task.taskExternalId} is missing startedAt or finishedAt`);
    }

    const meta = (task.additionalMetadata || {}) as Record<string, string>;
    const startedAt = toMicros(task.startedAt);
    const finishedAt = toMicros(task.finishedAt);
    const existing = windows[task.workflowRunExternalId];

    if (existing) {
      existing.startedAt = Math.min(existing.startedAt, startedAt);
      existing.finishedAt = Math.max(existing.finishedAt, finishedAt);
    } else {
      windows[task.workflowRunExternalId] = {
        runId: task.workflowRunExternalId,
        name: meta.name ?? '',
        digit: meta.digit ?? '',
        startedAt,
        finishedAt,
      };
    }
  }

  return Object.values(windows);
}

// Sweeps over the start and finish instants of the windows and returns the peak
// number of windows that were open at the same time, per key. A window that
// starts at the exact same microsecond another one finishes is a handoff, not
// an overlap.
function peakConcurrencyByKey(
  windows: RunWindow[],
  keyOf: (window: RunWindow) => string
): Record<string, number> {
  const peaks: Record<string, number> = {};
  const byKey: Record<string, RunWindow[]> = {};

  for (const window of windows) {
    const key = keyOf(window);
    byKey[key] = byKey[key] || [];
    byKey[key].push(window);
  }

  for (const [key, keyWindows] of Object.entries(byKey)) {
    const events = keyWindows.flatMap((window) => [
      { at: window.startedAt, delta: 1 },
      { at: window.finishedAt, delta: -1 },
    ]);
    events.sort((a, b) => a.at - b.at || a.delta - b.delta);

    let open = 0;
    let peak = 0;
    for (const event of events) {
      open += event.delta;
      peak = Math.max(peak, open);
    }
    peaks[key] = peak;
  }

  return peaks;
}

function violations(peaks: Record<string, number>, limit: number): Record<string, number> {
  return Object.fromEntries(Object.entries(peaks).filter(([, peak]) => peak > limit));
}

describe('concurrency-workflow-level-e2e', () => {
  const hatchet = makeE2EClient();

  it('workflow-level concurrency limits runs per digit and name', async () => {
    const testRunId = crypto.randomUUID();

    const runRefs = await Promise.all(
      Array.from({ length: NUM_RUNS }, () => {
        const name = pick(CHARACTERS);
        const digit = String(Math.floor(Math.random() * 6));
        return concurrencyWorkflowLevelWorkflow.runNoWait(
          { name, digit },
          {
            additionalMetadata: {
              test_run_id: testRunId,
              key: `${name}-${digit}`,
              name,
              digit,
            },
          }
        );
      })
    );

    await Promise.all(runRefs.map((ref) => ref.output));

    // The OLAP rows are written asynchronously, so wait until every task of
    // every run has both timestamps before measuring.
    const tasks = await poll(
      async () => {
        const resp = await hatchet.runs.list({
          workflowNames: [concurrencyWorkflowLevelWorkflow.name],
          additionalMetadata: { test_run_id: testRunId },
          limit: 1000,
          onlyTasks: true,
        });
        return resp.rows || [];
      },
      {
        timeoutMs: 60_000,
        intervalMs: 500,
        shouldStop: (rows) =>
          rows.length === NUM_RUNS * TASKS_PER_RUN &&
          rows.every((row) => !!row.startedAt && !!row.finishedAt),
        label: 'task rows with timestamps',
      }
    );

    const windows = buildRunWindows(tasks);
    expect(windows).toHaveLength(NUM_RUNS);

    const peakByDigit = peakConcurrencyByKey(windows, (window) => window.digit);
    const peakByName = peakConcurrencyByKey(windows, (window) => window.name);

    expect(violations(peakByDigit, DIGIT_MAX_RUNS)).toEqual({});
    expect(violations(peakByName, NAME_MAX_RUNS)).toEqual({});
  }, 240_000); // 100 runs with concurrency limits are slow in CI
});
