import { hatchet } from '../hatchet-client';
import { simple } from './workflow';

async function main() {
  const worker = await hatchet.worker('simple-worker', {
    workflows: [simple],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
