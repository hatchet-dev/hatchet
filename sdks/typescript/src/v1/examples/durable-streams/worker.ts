import { hatchet } from '../hatchet-client';
import { progressTask } from './workflow';

async function main() {
  const worker = await hatchet.worker('durable-streams-worker', {
    workflows: [progressTask],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
