import { hatchet } from '../../hatchet-client';
import { llmWf } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('llm-pipeline-worker', {
    workflows: [llmWf],
  });
  await worker.start();
  // !!
}

if (require.main === module) {
  main();
}
