import { hatchet } from '../../hatchet-client';
import { contentTask, safetyTask, evaluateTask, sectioningTask, votingTask } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('parallelization-worker', {
    workflows: [contentTask, safetyTask, evaluateTask, sectioningTask, votingTask],
    slots: 10,
  });
  await worker.start();
}

if (require.main === module) {
  main();
}
