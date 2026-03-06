import { hatchet } from '../../hatchet-client';
import { parentTask, childTask } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('batch-worker', {
    workflows: [parentTask, childTask],
    slots: 20,
  });
  await worker.start();
}

if (require.main === module) {
  main();
}
