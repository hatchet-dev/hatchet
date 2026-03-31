// > Declaring a Worker
import { hatchet } from '../hatchet-client';
import { getTemperature } from './workflow';

async function main() {
  const worker = await hatchet.worker('temperature-worker', {
    workflows: [getTemperature],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
// !!
