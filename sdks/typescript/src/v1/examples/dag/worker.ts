import { hatchet } from '../hatchet-client';
import { dag } from './workflow';

async function main() {
  const worker = await hatchet.worker('dag-worker', {
    workflows: [dag],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
