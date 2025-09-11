import { WorkflowRun } from '@hatchet/clients/admin';
import Hatchet from '../sdk';

const hatchet = Hatchet.init();

async function main() {
  const workflowRuns: WorkflowRun[] = [];

  workflowRuns[0] = {
    workflowName: 'bulk-parent-workflow',
    input: {},
    options: {
      additionalMetadata: {
        key: 'value',
      },
    },
  };

  workflowRuns[1] = {
    workflowName: 'bulk-parent-workflow',
    input: { second: 'second' },
    options: {
      additionalMetadata: {
        key: 'value',
      },
    },
  };

  try {
    const workflowRunResponse = hatchet.admin.runWorkflows(workflowRuns);

    const result = await workflowRunResponse;

    console.log('result', result);

    result.forEach(async (workflowRun) => {
      const stream = await workflowRun.stream();

      for await (const event of stream) {
        console.log('event received', event);
      }
    });
  } catch (error) {
    console.log('error', error);
  }
}

main();
