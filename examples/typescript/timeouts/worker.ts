// ❓ Declaring a Worker
import { hatchet } from '../hatchet-client';
import { cancellation } from './workflow';

async function main() {
  const worker = await hatchet.worker('cancellation-worker', {
    // 👀 Declare the workflows that the worker can execute
    workflows: [cancellation],
    // 👀 Declare the number of concurrent task runs the worker can accept
    slots: 100,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
