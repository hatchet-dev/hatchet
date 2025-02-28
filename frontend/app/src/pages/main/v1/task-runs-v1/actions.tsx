import { Button } from '@/components/v1/ui/button';
import api, {
  V1CancelTaskRequest,
  V1ReplayTaskRequest,
  V1TaskStatus,
} from '@/lib/api';
import { useTenant } from '@/lib/atoms';
import { useApiError } from '@/lib/hooks';
import { ArrowPathIcon, XCircleIcon } from '@heroicons/react/24/outline';
import { useMutation } from '@tanstack/react-query';
import { useCallback } from 'react';
import invariant from 'tiny-invariant';

export const TASK_RUN_TERMINAL_STATUSES = [
  V1TaskStatus.CANCELLED,
  V1TaskStatus.FAILED,
  V1TaskStatus.COMPLETED,
];

type ActionType = 'cancel' | 'replay';

type TaskRunActionsParams =
  | {
      actionType: 'cancel';
      filter?: never;
      externalIds: NonNullable<V1CancelTaskRequest['externalIds']>;
    }
  | {
      actionType: 'cancel';
      filter: NonNullable<V1CancelTaskRequest['filter']>;
      externalIds?: never;
    }
  | {
      actionType: 'replay';
      filter?: never;
      externalIds: NonNullable<V1ReplayTaskRequest['externalIds']>;
    }
  | {
      actionType: 'replay';
      filter: NonNullable<V1ReplayTaskRequest['filter']>;
      externalIds?: never;
    };

export const useTaskRunActions = () => {
  const { tenant } = useTenant();

  invariant(tenant?.metadata.id);

  const { handleApiError } = useApiError({});

  const { mutate: handleAction } = useMutation({
    mutationKey: ['task-run:action'],
    mutationFn: async (params: TaskRunActionsParams) => {
      const actionType: ActionType = params.actionType;

      switch (actionType) {
        case 'cancel':
          return await api.v1TaskCancel(tenant.metadata.id, params);
        case 'replay':
          return await api.v1TaskReplay(tenant.metadata.id, params);
        default:
          const exhaustiveCheck: never = actionType;
          throw new Error(`Unhandled action type: ${exhaustiveCheck}`);
      }
    },
    onError: handleApiError,
  });

  const handleTaskRunAction = useCallback(
    (params: TaskRunActionsParams) => {
      handleAction(params);
    },
    [handleAction],
  );

  return { handleTaskRunAction };
};

const BaseActionButton = ({
  disabled,
  onClick,
  icon,
  label,
}: {
  disabled: boolean;
  onClick: () => void;
  icon: JSX.Element;
  label: string;
}) => {
  return (
    <Button
      size={'sm'}
      className="px-2 py-2 gap-2"
      variant={'outline'}
      disabled={disabled}
      onClick={onClick}
    >
      {icon}
      {label}
    </Button>
  );
};

export const TaskRunActionButton = ({
  actionType,
  disabled,
  handleAction,
}: {
  actionType: ActionType;
  disabled: boolean;
  handleAction: () => void;
}) => {
  switch (actionType) {
    case 'cancel':
      return (
        <BaseActionButton
          disabled={disabled}
          onClick={handleAction}
          icon={<XCircleIcon className="w-4 h-4" />}
          label={'Cancel'}
        />
      );
    case 'replay':
      return (
        <BaseActionButton
          disabled={disabled}
          onClick={handleAction}
          icon={<ArrowPathIcon className="w-4 h-4" />}
          label={'Replay'}
        />
      );
    default:
      const exhaustiveCheck: never = actionType;
      throw new Error(`Unhandled action type: ${exhaustiveCheck}`);
  }
};
