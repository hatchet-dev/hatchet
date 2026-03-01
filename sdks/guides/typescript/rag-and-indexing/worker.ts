import { hatchet } from '../../hatchet-client';
import { ragWf } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('rag-worker', {
    workflows: [ragWf],
  });
  await worker.start();
  // !!
}

if (require.main === module) {
  main();
}
