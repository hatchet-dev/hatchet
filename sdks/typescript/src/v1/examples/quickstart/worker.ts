import { hatchet } from '../hatchet-client';
import { firstTask } from './workflows/first-task';

async function main() {
  const worker = await hatchet.worker('first-worker', {
    workflows: [firstTask],
    slots: 10,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
