import { hatchet } from '../hatchet-client';
import { dagWithConditions } from './workflow';

async function main() {
  const worker = await hatchet.worker('dag-worker', {
    workflows: [dagWithConditions],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
