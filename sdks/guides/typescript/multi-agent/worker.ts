import { hatchet } from '../../hatchet-client';
import { researchTask, writingTask, codeTask, orchestrator } from './workflow';

async function main() {
  // > Step 03 Run Worker
  const worker = await hatchet.worker('multi-agent-worker', {
    workflows: [researchTask, writingTask, codeTask, orchestrator],
    slots: 10,
  });
  await worker.start();
  // !!
}

if (require.main === module) {
  main();
}
