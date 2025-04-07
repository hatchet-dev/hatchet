import { hatchet } from '../hatchet-client';
import { onCron } from './workflow';

async function main() {
  const worker = await hatchet.worker('on-cron-worker', {
    register: [onCron],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
