import { hatchet } from '../client';
import { durableSleep } from './workflow';

async function main() {
  const worker = await hatchet.worker('sleep-worker', {
    workflows: [durableSleep],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
