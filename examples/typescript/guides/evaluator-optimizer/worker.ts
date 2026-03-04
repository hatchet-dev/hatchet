import { hatchet } from '../../hatchet-client';
import { generatorTask, evaluatorTask, optimizerTask } from './workflow';

async function main() {
  // > Step 03 Run Worker
  const worker = await hatchet.worker('evaluator-optimizer-worker', {
    workflows: [generatorTask, evaluatorTask, optimizerTask],
    slots: 5,
  });
  await worker.start();
}

if (require.main === module) {
  main();
}
