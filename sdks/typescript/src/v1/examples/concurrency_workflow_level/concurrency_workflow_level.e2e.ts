import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';
import { concurrencyWorkflowLevelWorkflow, DIGIT_MAX_RUNS, NAME_MAX_RUNS } from './workflow';

const CHARACTERS = ['Anna', 'Vronsky', 'Stiva', 'Dolly', 'Levin', 'Karenin'] as const;

function pick<T>(arr: readonly T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

type RunMetadata = {
  testRunId: string;
  key: string;
  name: string;
  digit: string;
  startedAt: Date;
  finishedAt: Date;
};

function parseRunMetadata(task: any): RunMetadata {
  const meta = task.additionalMetadata || {};
  return {
    testRunId: meta.test_run_id ?? '',
    key: meta.key ?? '',
    name: meta.name ?? '',
    digit: meta.digit ?? '',
    startedAt: task.startedAt ? new Date(task.startedAt) : new Date(8640000000000000),
    finishedAt: task.finishedAt ? new Date(task.finishedAt) : new Date(-8640000000000000),
  };
}

function areOverlapping(x: RunMetadata, y: RunMetadata): boolean {
  return (
    (x.startedAt.getTime() < y.finishedAt.getTime() &&
      x.finishedAt.getTime() > y.startedAt.getTime()) ||
    (x.finishedAt.getTime() > y.startedAt.getTime() &&
      x.startedAt.getTime() < y.finishedAt.getTime())
  );
}

function isValidGroup(group: RunMetadata[]): boolean {
  const digits: Record<string, number> = {};
  const names: Record<string, number> = {};

  for (const task of group) {
    digits[task.digit] = (digits[task.digit] || 0) + 1;
    names[task.name] = (names[task.name] || 0) + 1;
  }

  if (Object.values(digits).some((v) => v > DIGIT_MAX_RUNS)) return false;
  if (Object.values(names).some((v) => v > NAME_MAX_RUNS)) return false;

  return true;
}

describe('concurrency-workflow-level-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'concurrency-workflow-level-e2e-worker',
      workflows: [concurrencyWorkflowLevelWorkflow],
      slots: 10,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it('workflow-level concurrency limits runs per digit and name', async () => {
    const testRunId = crypto.randomUUID();

    const runRefs = await Promise.all(
      Array.from({ length: 100 }, () => {
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

    const runsResp = await hatchet.runs.list({
      workflowNames: [concurrencyWorkflowLevelWorkflow.name],
      additionalMetadata: { test_run_id: testRunId },
      limit: 1000,
      onlyTasks: false,
    });
    const runs = runsResp.rows || [];

    const sortedRuns = runs
      .map(parseRunMetadata)
      .sort((a, b) => a.startedAt.getTime() - b.startedAt.getTime());

    const overlappingGroups: Record<number, RunMetadata[]> = {};

    for (const run of sortedRuns) {
      let hasGroupMembership = false;

      if (Object.keys(overlappingGroups).length === 0) {
        overlappingGroups[1] = [run];
      } else {
        for (const group of Object.values(overlappingGroups)) {
          if (group.every((task) => areOverlapping(run, task))) {
            group.push(run);
            hasGroupMembership = true;
            break;
          }
        }

        if (!hasGroupMembership) {
          overlappingGroups[Object.keys(overlappingGroups).length + 1] = [run];
        }
      }
    }

    for (const group of Object.values(overlappingGroups)) {
      expect(isValidGroup(group)).toBe(true);
    }
  }, 240_000); // 100 runs with concurrency limits are slow in CI
});
