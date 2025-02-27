import { Button } from '@/components/v1/ui/button';
import api, { V1CancelTaskRequest, V1TaskStatus } from '@/lib/api';
import { useTenant } from '@/lib/atoms';
import { useApiError } from '@/lib/hooks';
import { XCircleIcon } from '@heroicons/react/24/outline';
import { useMutation } from '@tanstack/react-query';
import { useCallback } from 'react';
import invariant from 'tiny-invariant';

export const TASK_RUN_TERMINAL_STATUSES = [
  V1TaskStatus.CANCELLED,
  V1TaskStatus.FAILED,
  V1TaskStatus.COMPLETED,
];

type TaskRunCancellationParams =
  | {
      filter?: never;
      externalIds: V1CancelTaskRequest['externalIds'];
    }
  | {
      filter: V1CancelTaskRequest['filter'];
      externalIds?: never;
    };

export const useCancelTaskRuns = () => {
  const { tenant } = useTenant();

  invariant(tenant?.metadata.id);

  const { handleApiError } = useApiError({});

  const { mutate: cancelTaskRun } = useMutation({
    mutationKey: ['task-run:cancel'],
    mutationFn: async (params: TaskRunCancellationParams) => {
      await api.v1TaskCancel(tenant.metadata.id, params);
    },
    onSuccess: () => {
      console.log('Success');
    },
    onError: handleApiError,
  });

  const handleCancelTaskRun = useCallback(
    (params: TaskRunCancellationParams) => {
      cancelTaskRun(params);
    },
    [],
  );

  return { handleCancelTaskRun };
};

export const CancelTaskRunButton = ({
  disabled,
  params,
}: {
  disabled: boolean;
  params: TaskRunCancellationParams;
}) => {
  const { handleCancelTaskRun } = useCancelTaskRuns();

  return (
    <Button
      size={'sm'}
      className="px-2 py-2 gap-2"
      variant={'outline'}
      disabled={disabled}
      onClick={() => handleCancelTaskRun(params)}
    >
      <XCircleIcon className="w-4 h-4" />
      Cancel
    </Button>
  );
};
