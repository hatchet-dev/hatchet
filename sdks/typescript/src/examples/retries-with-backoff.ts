import Hatchet from '../sdk';
import { Workflow } from '../workflow';

const hatchet = Hatchet.init();

let numRetries = 0;

// > Backoff
const workflow: Workflow = {
  // ... normal workflow definition
  id: 'retries-with-backoff',
  description: 'Backoff',
  // ,
  steps: [
    {
      name: 'backoff-step',
      // ... step definition
      run: async (ctx) => {
        if (numRetries < 5) {
          numRetries += 1;
          throw new Error('failed');
        }

        return { backoff: 'completed' };
      },
      // ,
      retries: 10,
      // 👀 Backoff configuration
      backoff: {
        // 👀 Maximum number of seconds to wait between retries
        maxSeconds: 60,
        // 👀 Factor to increase the wait time between retries.
        // This sequence will be 2s, 4s, 8s, 16s, 32s, 60s... due to the maxSeconds limit
        factor: 2,
      },
    },
  ],
};
// !!

async function main() {
  const worker = await hatchet.worker('backoff-worker');
  await worker.registerWorkflow(workflow);
  worker.start();
}

main();
