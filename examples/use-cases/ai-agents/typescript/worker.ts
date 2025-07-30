/**
 * Worker for AI Agent workflow
 */

import { Hatchet } from '@hatchet-dev/typescript-sdk/v1';
import aiAgentWorkflow from './ai-agent';

const hatchet = new Hatchet();

async function main() {
  const worker = await hatchet.worker('ai-agent-worker', {
    workflows: [aiAgentWorkflow],
    slots: 10
  });

  console.log('AI Agent worker started');
  await worker.start();
}

if (require.main === module) {
  main().catch(console.error);
}