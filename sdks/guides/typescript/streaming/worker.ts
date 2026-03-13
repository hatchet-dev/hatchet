import { hatchet } from '../../hatchet-client';
import { streamTask } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('streaming-worker', {
    workflows: [streamTask],
  });
  await worker.start();
  // !!
}

if (require.main === module) {
  main();
}
