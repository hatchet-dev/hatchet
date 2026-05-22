import { hatchet } from '../hatchet-client';
import { tuningEnginesWorkflow } from './workflow';

async function main() {
  const worker = await hatchet.worker('tuning-engines-worker', {
    workflows: [tuningEnginesWorkflow],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
