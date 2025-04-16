// ❓ Declaring a Worker
import { hatchet } from '../hatchet-client';
import { child } from './workflow-with-child';

async function main() {
  const worker = await hatchet.worker('child-worker', {
    // 👀 Declare the workflows that the worker can execute
    workflows: [child],
    // 👀 Declare the number of concurrent task runs the worker can accept
    slots: 1000,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
// !!
