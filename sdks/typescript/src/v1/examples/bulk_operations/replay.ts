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

  // > Replay by run ids
  const runIds = runs.rows!.map((run) => run.metadata.id);

  await hatchet.runs.replay({ ids: runIds });
  // !!

  // > Replay by filters
  await hatchet.runs.replay({
    filters: {
      since: new Date(Date.now() - 24 * 60 * 60 * 1000),
      until: new Date(),
      statuses: [V1TaskStatus.FAILED],
      workflowNames: [workflow.name!],
      additionalMetadata: { key: 'value' },
    },
  });
  // !!
}
