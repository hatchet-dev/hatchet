import { Button } from '@/components/v1/ui/button';
import api, { V1ReplayTaskRequest, V1TaskStatus } from '@/lib/api';
import { useTenant } from '@/lib/atoms';
import { useApiError } from '@/lib/hooks';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { useMutation } from '@tanstack/react-query';
import { useCallback } from 'react';
import invariant from 'tiny-invariant';

export const TASK_RUN_TERMINAL_STATUSES = [
  V1TaskStatus.CANCELLED,
  V1TaskStatus.FAILED,
  V1TaskStatus.COMPLETED,
];

type TaskRunReplayParams =
  | {
      filter?: never;
      externalIds: NonNullable<V1ReplayTaskRequest['externalIds']>;
    }
  | {
      filter: NonNullable<V1ReplayTaskRequest['filter']>;
      externalIds?: never;
    };

export const useReplayTaskRuns = () => {
  const { tenant } = useTenant();

  invariant(tenant?.metadata.id);

  const { handleApiError } = useApiError({});

  const { mutate: replayTaskRun } = useMutation({
    mutationKey: ['task-run:replay'],
    mutationFn: async (params: TaskRunReplayParams) => {
      await api.v1TaskReplay(tenant.metadata.id, params);
    },
    onError: handleApiError,
  });

  const handleReplayTaskRun = useCallback(
    (params: TaskRunReplayParams) => {
      replayTaskRun(params);
    },
    [replayTaskRun],
  );

  return { handleReplayTaskRun };
};

export const ReplayTaskRunButton = ({
  disabled,
  handleReplayTaskRun,
}: {
  disabled: boolean;
  handleReplayTaskRun: () => void;
}) => {
  return (
    <Button
      size={'sm'}
      className="px-2 py-2 gap-2"
      variant={'outline'}
      onClick={handleReplayTaskRun}
      disabled={disabled}
    >
      <ArrowPathIcon className="w-4 h-4" />
      Replay
    </Button>
  );
};
