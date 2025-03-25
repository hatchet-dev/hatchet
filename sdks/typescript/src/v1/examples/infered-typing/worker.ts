import { hatchet } from '../client';
import { declaredType, inferredType, inferredTypeDurable, crazyWorkflow } from './workflow';

async function main() {
  const worker = await hatchet.worker('simple-worker', {
    workflows: [declaredType, inferredType, inferredTypeDurable, crazyWorkflow],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
