import { hatchet } from '../client';
import { simple } from './workflow';

async function main() {
  const worker = await hatchet.worker('healthcheck-worker', {
    workflows: [simple],
    healthcheck: true,
    slots: 10,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
