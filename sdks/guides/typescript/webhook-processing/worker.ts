import { hatchet } from '../../hatchet-client';
import { processWebhook } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('webhook-worker', {
    workflows: [processWebhook],
  });
  await worker.start();
  // !!
}

if (require.main === module) {
  main();
}
