import { hatchet } from '../../hatchet-client';
import { docWf } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('document-worker', {
    workflows: [docWf],
  });
  await worker.start();
}

if (require.main === module) {
  main();
}
