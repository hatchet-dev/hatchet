import sleep from '@hatchet/util/sleep';
import { WorkerList } from '@hatchet/clients/rest/generated/data-contracts';
import { checkDurableEvictionSupport, stopWorker } from '../__e2e__/harness';
import { Worker } from '../../client/worker/worker';
import { hatchet } from '../hatchet-client';
import { affinityExampleTask } from './workflow';

const labels = ['foo', 'bar'] as const;

describe('runtime-affinity-e2e', () => {
  let workerA: Worker | undefined;
  let workerB: Worker | undefined;
  let evictionSupported = false;

  beforeAll(async () => {
    evictionSupported = await checkDurableEvictionSupport(hatchet);
  });

  afterAll(async () => {
    await stopWorker(workerA);
    await stopWorker(workerB);
  });

  it('routes runs to the correct worker based on desired labels', async () => {
    if (!evictionSupported) {
      return;
    }

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
