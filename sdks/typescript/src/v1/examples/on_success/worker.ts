import { hatchet } from '../hatchet-client';
import { onSuccessDag } from './workflow';

async function main() {
  const worker = await hatchet.worker('always-succeed-worker', {
    register: [onSuccessDag],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
