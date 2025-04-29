import { hatchet } from '../hatchet-client';
import { priorityTasks } from './workflow';

async function main() {
  const worker = await hatchet.worker('priority-worker', {
    workflows: [...priorityTasks],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
