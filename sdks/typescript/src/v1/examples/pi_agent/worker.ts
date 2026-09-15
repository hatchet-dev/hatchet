import { hatchet } from '../hatchet-client';
import { piSessionManager } from './manager';
import { piTurn } from './turn';

async function main() {
  // > Register the worker
  const worker = await hatchet.worker('pi-agent-worker', {
    workflows: [piSessionManager, piTurn],
  });

  await worker.start();
  // !!
}

if (require.main === module) {
  main();
}
