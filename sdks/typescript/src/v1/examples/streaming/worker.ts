import { hatchet } from '../hatchet-client';
import { streamingTask } from './workflow';

async function main() {
  const worker = await hatchet.worker('streaming-worker', {
    workflows: [streamingTask],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
