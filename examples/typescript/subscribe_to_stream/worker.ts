import { hatchet } from '../hatchet-client';
import { dagStream, longStream } from './workflow';

async function main() {
  const worker = await hatchet.worker('subscribe-stream-worker', {
    workflows: [longStream, dagStream],
    slots: 8,
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
