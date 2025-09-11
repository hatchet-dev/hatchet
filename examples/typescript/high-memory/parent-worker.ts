// ❓ Declaring a Worker
import { hatchet } from '../hatchet-client';
import { parent } from './workflow-with-child';

async function main() {
  const worker = await hatchet.worker('parent-worker', {
    // 👀 Declare the workflows that the worker can execute
    workflows: [parent],
    // 👀 Declare the number of concurrent task runs the worker can accept
    slots: 20,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
