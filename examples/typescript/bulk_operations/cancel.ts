import { V1TaskStatus } from '@hatchet-dev/typescript-sdk/clients/rest/generated/data-contracts';
import { hatchet } from '../hatchet-client';

async function main() {
  // > Setup
  const workflows = await hatchet.workflows.list();

  if (!workflows.rows?.length) {
    throw new Error('no workflows found');
  }

  const [workflow] = workflows.rows;

  // > List runs
  const workflowRuns = await hatchet.runs.list({
    workflowNames: [workflow.name],
  });

  // > Cancel by run ids
  const runIds = workflowRuns.rows?.map((run) => run.metadata.id) ?? [];

  // to replay runs by their ids, use `hatchet.runs.replay` instead
  await hatchet.runs.cancel({ ids: runIds });

  // > Cancel by filters
  // to replay runs matching filters, use `hatchet.runs.replay` instead
  await hatchet.runs.cancel({
    filters: {
      since: new Date(Date.now() - 24 * 60 * 60 * 1000),
      until: new Date(),
      statuses: [V1TaskStatus.RUNNING],
      workflowNames: [workflow.name],
      additionalMetadata: { key: 'value' },
    },
  });
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}
