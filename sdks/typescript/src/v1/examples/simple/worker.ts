import { hatchet } from '../client';
import { simple } from './workflow';

async function main() {
  const worker = await hatchet.createWorker({
    workflows: [simple],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
