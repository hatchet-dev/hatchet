import { hatchet } from '../../hatchet-client';
import { approvalTask } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('human-in-the-loop-worker', {
    workflows: [approvalTask],
  });
  await worker.start();
}

if (require.main === module) {
  main();
}
