import { hatchet } from '../hatchet-client';
import { progressTask, retryingProgressTask } from './workflow';

async function main() {
  const worker = await hatchet.worker('durable-streams-worker', {
    workflows: [progressTask, retryingProgressTask],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
