import { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import { hatchet } from '../hatchet-client';

async function main() {
  const { runs } = hatchet;

  // > API operations
  // list all failed runs
  const allFailedRuns = await runs.list({
    statuses: [V1TaskStatus.FAILED],
  });

  // replay by ids
  await runs.replay({ ids: allFailedRuns.rows?.map((r) => r.metadata.id) });

  // or you can run bulk operations with filters directly
  await runs.cancel({
    filters: {
      since: new Date('2025-03-27'),
      additionalMetadata: { user: '123' },
    },
  });
  // !!
}
