import type { V1TaskSummary } from '@/lib/api/generated/data-contracts';

export function isStandaloneRun(
  run: Pick<V1TaskSummary, 'metadata' | 'workflowRunExternalId'>,
): boolean {
  return run.workflowRunExternalId === run.metadata.id;
}
