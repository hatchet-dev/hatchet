import { Button } from '@/components/v1/ui/button';
import {
  DialogTitle,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
} from '@/components/v1/ui/dialog';
import api, {
  V1CancelTaskRequest,
  V1ReplayTaskRequest,
  V1TaskStatus,
} from '@/lib/api';
import { useTenant } from '@/lib/atoms';
import { useApiError } from '@/lib/hooks';
import { ArrowPathIcon, XCircleIcon } from '@heroicons/react/24/outline';
import { useMutation } from '@tanstack/react-query';
import { useCallback, useState } from 'react';
import invariant from 'tiny-invariant';

export const TASK_RUN_TERMINAL_STATUSES = [
  V1TaskStatus.CANCELLED,
  V1TaskStatus.FAILED,
  V1TaskStatus.COMPLETED,
];

type ActionType = 'cancel' | 'replay';

type BaseTaskRunActionParams =
  | {
      filter?: never;
      externalIds:
        | NonNullable<V1CancelTaskRequest['externalIds']>
        | NonNullable<V1ReplayTaskRequest['externalIds']>;
    }
  | {
      filter:
        | NonNullable<V1CancelTaskRequest['filter']>
        | NonNullable<V1ReplayTaskRequest['filter']>;
      externalIds?: never;
    };

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
          return api.v1TaskCancel(tenant.metadata.id, params);
        case 'replay':
          return api.v1TaskReplay(tenant.metadata.id, params);
        default:
          // eslint-disable-next-line no-case-declarations
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

type ConfirmActionModalProps = {
  actionType: ActionType;
  isOpen: boolean;
  setIsOpen: (isOpen: boolean) => void;
  onConfirm: () => void;
  params: TaskRunActionsParams;
};

const actionTypeToLabel = (actionType: ActionType) => {
  switch (actionType) {
    case 'cancel':
      return 'Cancel';
    case 'replay':
      return 'Replay';
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = actionType;
      throw new Error(`Unhandled action type: ${exhaustiveCheck}`);
  }
};

type ModalContentProps = {
  label: string;
  params: TaskRunActionsParams;
};

const CancelByExternalIdsContent = ({ label, params }: ModalContentProps) => {
  return (
    <>
      <p className="text-lg">
        You're about to {label.toLowerCase()} the following task runs:
      </p>
      <ul className="list-disc pl-4 pt-2">
        {params.externalIds
          ?.slice(0, 10)
          .map((externalId) => <li key={externalId}>{externalId}</li>)}
        {(params.externalIds?.length || 0) > 10 && (
          <li>{(params.externalIds?.length || 0) - 10} more</li>
        )}
      </ul>
    </>
  );
};

type FilterName = keyof NonNullable<V1CancelTaskRequest['filter']>;

const taskFilterToLabel = (filter: FilterName) => {
  switch (filter) {
    case 'additionalMetadata':
      return 'Additional metadata';
    case 'since':
      return 'Since';
    case 'statuses':
      return 'Statuses';
    case 'until':
      return 'Until';
    case 'workflowIds':
      return 'Workflow IDs';
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = filter;
      throw new Error(`Unhandled filter: ${exhaustiveCheck}`);
  }
};

const ModalContent = ({ label, params }: ModalContentProps) => {
  if (params.externalIds?.length) {
    return <CancelByExternalIdsContent label={label} params={params} />;
  } else if (params.filter) {
    const kvs = Object.entries(params.filter).filter(([k, v]) => !!v && !!k);

    return (
      <>
        <p className="text-lg">
          You're about to {label.toLowerCase()} all task runs matching the
          following filters:
        </p>
        <ul className="list-disc pl-4 pt-2">
          {kvs.map(([k, v]) => (
            <li key={k}>{`${taskFilterToLabel(k as FilterName)}: ${v}`}</li>
          ))}
        </ul>
      </>
    );
  } else {
    throw new Error(`Unhandled case: ${params}`);
  }
};

const ConfirmActionModal = ({
  actionType,
  isOpen,
  setIsOpen,
  onConfirm,
  params,
}: ConfirmActionModalProps) => {
  const label = actionTypeToLabel(actionType);

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogContent className="sm:max-w-[625px] py-12 max-h-screen overflow-auto">
        <DialogHeader className="gap-2">
          <div className="flex flex-row justify-between items-center w-full">
            <DialogTitle>{label} task runs</DialogTitle>
          </div>
        </DialogHeader>

        <div className="flex flex-col mt-4">
          <DialogDescription>
            <ModalContent label={label} params={params} />
          </DialogDescription>

          <div className="flex flex-row items-center flex-1 gap-x-2 justify-end">
            <Button
              className="mt-6 w-full sm:w-auto sm:self-end"
              onClick={() => {
                setIsOpen(false);
              }}
              variant="outline"
            >
              Cancel
            </Button>{' '}
            <Button
              className="mt-6 w-full sm:w-auto sm:self-end"
              onClick={() => {
                onConfirm();
                setIsOpen(false);
              }}
            >
              Confirm
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

const BaseActionButton = ({
  disabled,
  params,
  icon,
  label,
  showModal,
}: {
  disabled: boolean;
  params: TaskRunActionsParams;
  icon: JSX.Element;
  label: string;
  showModal: boolean;
}) => {
  const [isConfirmModalOpen, setIsConfirmModalOpen] = useState(false);
  const { handleTaskRunAction } = useTaskRunActions();

  const handleAction = useCallback(() => {
    if (params.externalIds?.length) {
      handleTaskRunAction({
        actionType: params.actionType,
        externalIds: params.externalIds,
      });
    } else if (
      params.filter &&
      Object.values(params.filter).some((filter) => !!filter)
    ) {
      handleTaskRunAction({
        actionType: params.actionType,
        filter: params.filter,
      });
    }
  }, [handleTaskRunAction, params]);

  return (
    <>
      <ConfirmActionModal
        actionType={params.actionType}
        isOpen={isConfirmModalOpen}
        setIsOpen={setIsConfirmModalOpen}
        onConfirm={handleAction}
        params={params}
      />
      <Button
        size={'sm'}
        className="px-2 py-2 gap-2"
        variant={'outline'}
        disabled={disabled}
        onClick={() => {
          if (!showModal) {
            handleAction();
            return;
          }

          setIsConfirmModalOpen(true);
        }}
      >
        {icon}
        {label}
      </Button>
    </>
  );
};

export const TaskRunActionButton = ({
  actionType,
  disabled,
  params,
  showModal,
}: {
  actionType: ActionType;
  disabled: boolean;
  params: BaseTaskRunActionParams;
  showModal: boolean;
}) => {
  switch (actionType) {
    case 'cancel':
      return (
        <BaseActionButton
          disabled={disabled}
          params={{ ...params, actionType: 'cancel' }}
          icon={<XCircleIcon className="w-4 h-4" />}
          label={'Cancel'}
          showModal={showModal}
        />
      );
    case 'replay':
      return (
        <BaseActionButton
          disabled={disabled}
          params={{ ...params, actionType: 'replay' }}
          icon={<ArrowPathIcon className="w-4 h-4" />}
          label={'Replay'}
          showModal={showModal}
        />
      );
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = actionType;
      throw new Error(`Unhandled action type: ${exhaustiveCheck}`);
  }
};
