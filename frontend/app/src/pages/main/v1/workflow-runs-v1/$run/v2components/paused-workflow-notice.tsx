import { V1TaskStatus, queries } from '@/lib/api';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';

export function PausedWorkflowNotice({
  workflowId,
  status,
}: {
  workflowId: string;
  status: V1TaskStatus;
}) {
  const workflowQuery = useQuery({
    ...queries.workflows.get(workflowId),
    enabled: status === V1TaskStatus.QUEUED,
  });

  if (status !== V1TaskStatus.QUEUED || !workflowQuery.data?.isPaused) {
    return null;
  }

  return (
    <div className="mb-4 flex items-start gap-3 rounded-md border border-yellow-500/50 bg-yellow-500/10 px-4 py-3 text-sm">
      <ExclamationTriangleIcon className="mt-0.5 h-4 w-4 shrink-0 text-yellow-500" />
      <p className="text-yellow-600 dark:text-yellow-400">
        This run is queued because its workflow is currently paused. It will
        resume once the workflow is unpaused.
      </p>
    </div>
  );
}
