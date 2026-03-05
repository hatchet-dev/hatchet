import { hatchet } from '../../hatchet-client';
import { classifyTask, supportTask, salesTask, defaultTask, routerTask } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('routing-worker', {
    workflows: [classifyTask, supportTask, salesTask, defaultTask, routerTask],
    slots: 5,
  });
  await worker.start();
}

if (require.main === module) {
  main();
}
