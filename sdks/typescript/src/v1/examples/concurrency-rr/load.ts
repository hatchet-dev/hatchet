/* eslint-disable no-plusplus */
import { hatchet } from '../hatchet-client';
import { simpleConcurrency } from './workflow';

function generateRandomString(length: number): string {
  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += characters.charAt(Math.floor(Math.random() * characters.length));
  }
  return result;
}

async function main() {
  const groupCount = 2;
  const runsPerGroup = 20_000;
  const BATCH_SIZE = 400;

  const workflowRuns = [];
  for (let i = 0; i < groupCount; i++) {
    for (let j = 0; j < runsPerGroup; j++) {
      workflowRuns.push({
        workflowName: simpleConcurrency.definition.name,
        input: {
          Message: generateRandomString(10),
          GroupKey: `group-${i}`,
        },
      });
    }
  }

  // Shuffle the workflow runs array
  for (let i = workflowRuns.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [workflowRuns[i], workflowRuns[j]] = [workflowRuns[j], workflowRuns[i]];
  }

  // Process workflows in batches
  for (let i = 0; i < workflowRuns.length; i += BATCH_SIZE) {
    const batch = workflowRuns.slice(i, i + BATCH_SIZE);
    await hatchet.admin.runWorkflows(batch);
  }
}

if (require.main === module) {
  main().then(() => process.exit(0));
}
