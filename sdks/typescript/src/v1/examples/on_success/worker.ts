import { hatchet } from '../hatchet-client';
import { onSuccessDag, onSuccess } from './workflow';

async function main() {
  const worker = await hatchet.worker('always-succeed-worker', {
    workflows: [onSuccessDag, onSuccess],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
