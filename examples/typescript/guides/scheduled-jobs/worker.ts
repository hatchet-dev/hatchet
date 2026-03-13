import { hatchet } from '../../hatchet-client';
import { cronWf } from './workflow';

async function main() {
  // > Step 03 Run Worker
  const worker = await hatchet.worker('scheduled-worker', {
    workflows: [cronWf],
  });
  await worker.start();
}

if (require.main === module) {
  main();
}
