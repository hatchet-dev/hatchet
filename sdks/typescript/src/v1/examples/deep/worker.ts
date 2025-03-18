import { hatchet } from '../client';
import { parent, child1, child2, child3, child4, child5 } from './workflow';

async function main() {
  const worker = await hatchet.worker('simple-worker', {
    workflows: [parent, child1, child2, child3, child4, child5],
    slots: 5000,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
