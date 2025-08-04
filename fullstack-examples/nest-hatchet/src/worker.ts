// ❓ Declaring a Worker
import { simple } from './tasks/simple';
import { HatchetClient } from '@hatchet-dev/typescript-sdk';

async function main() {
  const hatchet = HatchetClient.init();

  const worker = await hatchet.worker('simple-worker', {
    workflows: [simple].map((task) => task(hatchet)),
  });

  await worker.start();
}

if (require.main === module) {
  void main();
}
// !!
