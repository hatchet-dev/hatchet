import { hatchet } from '../hatchet-client';
import { refreshTimeoutTask, timeoutTask } from './workflow';

async function main() {
  const worker = await hatchet.worker('timeout-worker', {
    workflows: [timeoutTask, refreshTimeoutTask],
    slots: 50,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
