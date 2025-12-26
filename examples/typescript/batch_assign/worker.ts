// > Declaring a Worker
import { hatchet } from '../hatchet-client';
import { batch } from './task';
import { dag } from './workflow';

async function main() {
  const worker = await hatchet.worker('simple-worker', {
    // 👀 Declare the workflows that the worker can execute
    workflows: [batch, dag],
    // 👀 Declare the number of concurrent task runs the worker can accept
    slots: 2,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
