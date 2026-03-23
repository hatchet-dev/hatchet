import { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import { hatchet } from '../hatchet-client';

async function main() {
  // > Setup
  const workflows = await hatchet.workflows.list();

  const [workflow] = workflows.rows!;
  // !!

  // > List runs
  const runs = await hatchet.runs.list({
    workflowNames: [workflow.name!],
  });
  // !!

  // > Cancel by run ids
  const runIds = runs.rows!.map((run) => run.metadata.id);

  await hatchet.runs.cancel({ ids: runIds });
  // !!

  // > Cancel by filters
  await hatchet.runs.cancel({
    filters: {
      since: new Date(Date.now() - 24 * 60 * 60 * 1000),
      until: new Date(),
      statuses: [V1TaskStatus.RUNNING],
      workflowNames: [workflow.name!],
      additionalMetadata: { key: 'value' },
    },
  });
  // !!
}
