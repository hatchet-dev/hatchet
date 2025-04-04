import { hatchet } from '../hatchet-client';
import { durableSleep } from './workflow';

async function main() {
  const worker = await hatchet.worker('sleep-worker', {
    register: [durableSleep],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
