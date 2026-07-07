import { hatchet } from '../hatchet-client';
import { getTemperature, getTemperatureWorkflow } from './workflow';

async function main() {
  const worker = await hatchet.worker('temperature-worker', {
    workflows: [getTemperature, getTemperatureWorkflow],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
