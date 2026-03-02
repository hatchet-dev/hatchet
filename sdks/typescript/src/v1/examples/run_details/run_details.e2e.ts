import { randomUUID } from 'crypto';
import { makeE2EClient, poll } from '../__e2e__/harness';
import { runDetailTestWorkflow } from './workflow';
import { V1TaskStatus } from '../../../clients/rest/generated/data-contracts';

describe('run-details-e2e', () => {
  const hatchet = makeE2EClient();

  xit('get run details mid-execution and after completion', async () => {
    const mockInput = { foo: randomUUID() };
    const testRunId = randomUUID();
    const meta = { test_run_id: testRunId };

    const ref = await runDetailTestWorkflow.runNoWait(mockInput, {
      additionalMetadata: meta,
    });

    let details = await poll(async () => hatchet.runs.get(ref), {
      timeoutMs: 10_000,
      intervalMs: 100,
      label: 'run has started',
      shouldStop: (d) =>
        [V1TaskStatus.RUNNING, V1TaskStatus.QUEUED, V1TaskStatus.FAILED].includes(
          d.run.status as any
        ),
    });

    expect([V1TaskStatus.RUNNING, V1TaskStatus.QUEUED]).toContain(details.run.status);
    expect(details.run.input).toEqual(mockInput);
    expect((details.run as any).additionalMetadata || {}).toMatchObject(meta);

    const tasksByDisplayName = (details.tasks || []).reduce(
      (acc: Record<string, (typeof details.tasks)[0]>, t) => {
        acc[t.displayName || ''] = t;
        return acc;
      },
      {}
    );

    expect(Object.keys(tasksByDisplayName).length).toBe(4);
    expect(tasksByDisplayName.step3).toBeUndefined();
    expect(tasksByDisplayName.step4).toBeUndefined();

    for (const task of Object.values(tasksByDisplayName)) {
      expect([V1TaskStatus.RUNNING, V1TaskStatus.QUEUED, V1TaskStatus.COMPLETED]).toContain(
        task.status
      );
    }

    await expect(ref.output).rejects.toBeDefined();

    details = await poll(async () => hatchet.runs.get(ref), {
      timeoutMs: 30_000,
      intervalMs: 100,
      label: 'run status FAILED',
      shouldStop: (d) => d.run.status === V1TaskStatus.FAILED,
    });

    expect(details.run.status).toBe(V1TaskStatus.FAILED);
    expect(details.run.input).toEqual(mockInput);
    expect((details.run as any).additionalMetadata || {}).toMatchObject(meta);
    expect(details.tasks).toHaveLength(6);

    const taskIdToName =
      (details.shape || []).reduce(
        (acc: Record<string, string>, s: { taskExternalId?: string; taskName?: string }) => {
          if (s.taskExternalId && s.taskName) acc[s.taskExternalId] = s.taskName;
          return acc;
        },
        {}
      ) || {};
    const tasksByName = (details.tasks || []).reduce(
      (acc: Record<string, (typeof details.tasks)[0]>, t) => {
        const name = taskIdToName[t.taskExternalId] ?? t.displayName ?? '';
        acc[name] = t;
        return acc;
      },
      {}
    );

    expect(tasksByName.step1?.status).toBe(V1TaskStatus.COMPLETED);
    expect(tasksByName.step2?.status).toBe(V1TaskStatus.COMPLETED);
    expect(tasksByName.step3?.status).toBe(V1TaskStatus.COMPLETED);
    expect(tasksByName.step4?.status).toBe(V1TaskStatus.COMPLETED);
    expect(tasksByName.fail_step?.status).toBe(V1TaskStatus.FAILED);
    expect(tasksByName.cancel_step?.status).toBe(V1TaskStatus.CANCELLED);

    const step1Output = tasksByName.step1?.output as { random_number?: number };
    const step2Output = tasksByName.step2?.output as { random_number?: number };
    const step3Output = tasksByName.step3?.output as { sum?: number };

    expect(step1Output?.random_number).toBeDefined();
    expect(step2Output?.random_number).toBeDefined();
    expect(step3Output?.sum).toBe(step1Output!.random_number! + step2Output!.random_number!);

    expect(tasksByName.step4?.output).toEqual({ step4: 'step4' });
    expect(tasksByName.fail_step?.errorMessage).toBeDefined();
    expect(tasksByName.fail_step?.output).toBeFalsy();
  }, 120_000);
});
