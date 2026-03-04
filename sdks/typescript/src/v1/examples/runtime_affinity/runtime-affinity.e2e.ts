import sleep from '@hatchet-dev/typescript-sdk/util/sleep';
import { WorkerList } from '@hatchet-dev/typescript-sdk/clients/rest/generated/data-contracts';
import { stopWorker } from '../__e2e__/harness';
import { Worker } from '../../client/worker/worker';
import { hatchet } from '../hatchet-client';
import { affinityExampleTask } from './workflow';

const labels = ['foo', 'bar'] as const;

describe('runtime-affinity-e2e', () => {
  let workerA: Worker | undefined;
  let workerB: Worker | undefined;

  afterAll(async () => {
    await stopWorker(workerA);
    await stopWorker(workerB);
  });

  it('routes runs to the correct worker based on desired labels', async () => {
    workerA = await hatchet.worker('runtime-affinity-worker', {
      workflows: [affinityExampleTask],
      labels: { affinity: labels[0] },
    });
    workerA.start().catch((err) => console.error('[affinity-test] workerA start error:', err));
    await workerA.waitUntilReady(10_000);

    workerB = await hatchet.worker('runtime-affinity-worker', {
      workflows: [affinityExampleTask],
      labels: { affinity: labels[1] },
    });
    workerB.start().catch((err) => console.error('[affinity-test] workerB start error:', err));
    await workerB.waitUntilReady(10_000);

    await sleep(5_000);

    const allWorkers: WorkerList = await hatchet.workers.list();
    const activeWorkers = (allWorkers.rows || []).filter(
      (w) => w.status === 'ACTIVE' && `${w.name}`.includes('runtime-affinity-worker')
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

    // eslint-disable-next-line no-plusplus
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

      expect(res.worker_id).toBe(workerLabelToId[targetWorker]);
    }
  }, 120_000);
});
