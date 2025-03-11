import { hatchet } from '../client';
import { lower } from './workflow';

async function main() {
  const worker = await hatchet.createWorker({
    workflows: [lower],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
