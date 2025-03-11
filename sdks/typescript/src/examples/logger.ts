import Hatchet from '../sdk';
import { Workflow } from '../workflow';

const hatchet = Hatchet.init({
  log_level: 'OFF',
});

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

const workflow: Workflow = {
  id: 'logger-example',
  description: 'test',
  on: {
    event: 'user:create',
  },
  steps: [
    {
      name: 'logger-step1',
      run: async (ctx) => {
        // log in a for loop
        // eslint-disable-next-line no-plusplus
        for (let i = 0; i < 10; i++) {
          ctx.log(`log message ${i}`);
          await sleep(200);
        }

        return { step1: 'completed step run' };
      },
    },
  ],
};

async function main() {
  const worker = await hatchet.worker('logger-worker', 1);
  await worker.registerWorkflow(workflow);
  worker.start();
}

main();
