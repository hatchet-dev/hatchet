import { makeE2EClient, stopWorker } from '../__e2e__/harness';
import { Worker } from '../../client/worker/worker';
import { affinityExampleTask } from './workflow';

const labels = ['foo', 'bar'] as const;

describe('runtime-affinity-e2e', () => {
  const hatchet = makeE2EClient();
  let workerA: Worker | undefined;
  let workerB: Worker | undefined;

  beforeAll(async () => {
    workerA = await hatchet.worker('runtime-affinity-worker', {
      workflows: [affinityExampleTask],
      labels: { affinity: labels[0] },
    });
    void workerA.start();
    await workerA.waitUntilReady(10_000);

    workerB = await hatchet.worker('runtime-affinity-worker', {
      workflows: [affinityExampleTask],
      labels: { affinity: labels[1] },
    });
    void workerB.start();
    await workerB.waitUntilReady(10_000);
  });

  afterAll(async () => {
    await stopWorker(workerA);
    await stopWorker(workerB);
  });

  it('routes runs to the correct worker based on desired labels', async () => {
    const allWorkers = await hatchet.workers.list();
    const activeWorkers = (allWorkers.rows || []).filter(
      (w: any) =>
        w.status === 'ACTIVE' &&
        `${w.name}`.includes('runtime-affinity-worker')
    );

    expect(activeWorkers.length).toBe(2);

    const workerLabelToId: Record<string, string> = {};
    for (const worker of activeWorkers) {
      for (const label of worker.labels || []) {
        if (label.key === 'affinity' && labels.includes(label.value as any)) {
          workerLabelToId[label.value!] = worker.metadata.id;
        }
      }
    }

    expect(Object.keys(workerLabelToId).sort()).toEqual([...labels].sort());

    for (let i = 0; i < 20; i++) {
      const targetWorker = labels[i % 2];
      const res = await affinityExampleTask.run(
        {},
        {
          desiredWorkerLabels: {
            affinity: {
              value: targetWorker,
              required: true,
            },
          },
        }
      );

      expect((res as any).worker_id).toBe(workerLabelToId[targetWorker]);
    }
  }, 120_000);
});
