import sleep from '@hatchet/util/sleep';
import Hatchet, { Worker as Worker } from '../..';
import { simpleCronWorkflow } from './cron-worker';

xdescribe('cron-e2e', () => {
  fit(
    'should invoke the workflow on the cron schedule',
    async () => {
      let worker: Worker | undefined;
      try {
        const hatchet = Hatchet.init();
        worker = await hatchet.worker('example-worker');

        const startTime = new Date();

        await worker.registerWorkflow(simpleCronWorkflow);
        void worker.start();
        await sleep(60 * 2 + 1000);

        const workflowRuns = await hatchet.api.workflowRunList(hatchet.tenantId, {
          createdAfter: startTime.toISOString(),
        });

        expect(workflowRuns.data.rows?.length).toEqual(2);
      } finally {
        if (worker) {
          await worker.stop();
        }
      }
    },
    60 * 2 + 2000
  );
});
