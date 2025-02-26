import { WorkflowRun } from '@hatchet/clients/admin';
import Hatchet from '../sdk';

const hatchet = Hatchet.init();

async function main() {
  const workflowRuns: WorkflowRun[] = [];

  for (let i = 0; i < 100; i += 1) {
    workflowRuns.push({
      workflowName: 'simple-workflow',
      input: {},
      options: {
        additionalMetadata: {
          key: 'value',
          dedupe: 'key',
        },
      },
    });
  }

  const workflowRunResponse = hatchet.admin.runWorkflows(workflowRuns);

  const result = await workflowRunResponse;

  console.log('result', result);

  result.forEach(async (workflowRun) => {
    const stream = await workflowRun.stream();

    for await (const event of stream) {
      console.log('event received', event);
    }
  });
}

main();
