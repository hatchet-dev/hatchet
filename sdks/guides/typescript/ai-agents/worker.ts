import { hatchet } from '../../hatchet-client';
import { agentTask, streamingAgentTask } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('agent-worker', {
    workflows: [agentTask, streamingAgentTask],
    slots: 5,
  });
  await worker.start();
  // !!
}

if (require.main === module) {
  main();
}
