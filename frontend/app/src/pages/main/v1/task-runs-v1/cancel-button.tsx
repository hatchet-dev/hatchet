import { Button } from '@/components/v1/ui/button';
import api, { queries, V1CancelTaskRequest, V1TaskStatus } from '@/lib/api';
import { useTenant } from '@/lib/atoms';
import { useApiError } from '@/lib/hooks';
import { XCircleIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useCallback } from 'react';
import { useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';

const TASK_RUN_TERMINAL_STATUSES = [
  V1TaskStatus.CANCELLED,
  V1TaskStatus.FAILED,
  V1TaskStatus.COMPLETED,
];

const useTaskRun = ({ taskRunId }: { taskRunId: string }) => {
  const { tenant } = useTenant();

  const tenantId = tenant?.metadata.id;
  invariant(tenantId);

  const taskRunQuery = useQuery({
    ...queries.v1Tasks.get(taskRunId),
    refetchInterval: 5000,
  });

  const taskRun = taskRunQuery.data;
  invariant(taskRun);

  return {
    taskRun,
    ...taskRunQuery,
  };
};

export const CancelTaskRunButton = (data: V1CancelTaskRequest) => {
  const { run: taskRunId } = useParams();
  const { tenant } = useTenant();

  invariant(taskRunId);
  invariant(tenant?.metadata.id);

  const { taskRun } = useTaskRun({ taskRunId });

  const { handleApiError } = useApiError({});

  const { mutate: cancelTaskRun } = useMutation({
    mutationKey: ['task-run:cancel'],
    mutationFn: async () => {
      await api.v1TaskCancel(tenant.metadata.id, data);
    },
    onSuccess: () => {
      console.log('Success');
    },
    onError: handleApiError,
  });

  const handleCancelTaskRun = useCallback(() => {
    cancelTaskRun();
  }, []);

  return (
    <Button
      size={'sm'}
      className="px-2 py-2 gap-2"
      variant={'outline'}
      disabled={TASK_RUN_TERMINAL_STATUSES.includes(taskRun.status)}
      onClick={handleCancelTaskRun}
    >
      <XCircleIcon className="w-4 h-4" />
      Cancel
    </Button>
  );
};
