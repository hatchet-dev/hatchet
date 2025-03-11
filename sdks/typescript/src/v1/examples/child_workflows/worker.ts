import { hatchet } from '../client';
import { parent, child } from './workflow';

async function main() {
  const worker = await hatchet.createWorker({
    workflows: [parent, child],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
