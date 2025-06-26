import { hatchet } from '../hatchet-client';
import { streaming_task } from './workflow';


async function main() {
  const worker = await hatchet.worker('streaming-worker', {
    workflows: [streaming_task],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
