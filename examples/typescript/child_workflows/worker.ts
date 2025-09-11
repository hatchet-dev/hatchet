import { hatchet } from '../hatchet-client';
import { parent, child } from './workflow';

async function main() {
  const worker = await hatchet.worker('child-workflow-worker', {
    workflows: [parent, child],
    slots: 100,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
