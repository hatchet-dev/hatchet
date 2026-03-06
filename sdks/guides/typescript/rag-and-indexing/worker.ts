import { hatchet } from '../../hatchet-client';
import { ragWf, embedChunkTask, queryTask } from './workflow';

async function main() {
  // > Step 06 Run Worker
  const worker = await hatchet.worker('rag-worker', {
    workflows: [ragWf, embedChunkTask, queryTask],
  });
  await worker.start();
  // !!
}

if (require.main === module) {
  main();
}
