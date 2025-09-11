import { Button } from '@/components/v1/ui/button';
import {
  DialogTitle,
  Dialog,
  DialogContent,
  DialogHeader,
} from '@/components/v1/ui/dialog';
import api, {
  queries,
  V1CancelTaskRequest,
  V1ReplayTaskRequest,
  V1TaskStatus,
} from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { XCircleIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useCallback } from 'react';
import { Combobox } from '@/components/v1/molecules/combobox/combobox';
import {
  additionalMetadataKey,
  statusKey,
  workflowKey,
} from '../workflow-runs-v1/components/v1/task-runs-columns';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useRunsContext } from '../workflow-runs-v1/hooks/runs-provider';
import { TimeFilter } from '../workflow-runs-v1/components/task-runs-table/time-filter';
import { cn } from '@/lib/utils';
import { Repeat1 } from 'lucide-react';
import { useToast } from '@/components/v1/hooks/use-toast';
import { capitalize } from 'lodash';

export const TASK_RUN_TERMINAL_STATUSES = [
  V1TaskStatus.CANCELLED,
  V1TaskStatus.FAILED,
  V1TaskStatus.COMPLETED,
];

export type ActionType = 'cancel' | 'replay';

export type BaseTaskRunActionParams =
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

export type TaskRunActionsParams =
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
  const { tenantId } = useCurrentTenantId();
  const { toast } = useToast();

  const { handleApiError } = useApiError({});

  const onActionSubmit = useCallback(
    (actionType: ActionType) => {
      toast({
        title: `${capitalize(actionType)} request submitted`,
        description: `No need to hit '${capitalize(actionType)}' again.`,
      });
    },
    [toast],
  );

  const handleActionProcessed = useCallback(
    (action: 'cancel' | 'replay', ids: string[]) => {
      const prefix = action === 'cancel' ? 'Canceling' : 'Replaying';
      const count = ids.length;

      const t = toast({
        title: `${prefix} ${count} task run${count > 1 ? 's' : ''}`,
        description: `This may take a few seconds. You don't need to hit ${action} again.`,
      });

      setTimeout(() => {
        t.dismiss();
      }, 5000);
    },
    [toast],
  );

  const { mutateAsync: handleAction } = useMutation({
    mutationKey: ['task-run:action'],
    mutationFn: async (params: TaskRunActionsParams) => {
      const actionType: ActionType = params.actionType;

      switch (actionType) {
        case 'cancel':
          return api.v1TaskCancel(tenantId, params);
        case 'replay':
          return api.v1TaskReplay(tenantId, params);
        default:
          // eslint-disable-next-line no-case-declarations
          const exhaustiveCheck: never = actionType;
          throw new Error(`Unhandled action type: ${exhaustiveCheck}`);
      }
    },
    onError: handleApiError,
    onMutate: (params) => {
      onActionSubmit(params.actionType);
    },
  });

  const handleTaskRunActionInner = useCallback(
    async (params: TaskRunActionsParams) => {
      const resp = await handleAction(params);

      if (resp.data?.ids) {
        handleActionProcessed(params.actionType, resp.data.ids);
      }
    },
    [handleAction, handleActionProcessed],
  );

  const handleTaskRunAction = useCallback(
    (params: TaskRunActionsParams) => {
      if (params.externalIds?.length) {
        handleTaskRunActionInner({
          actionType: params.actionType,
          externalIds: params.externalIds,
        });
      } else if (
        params.filter &&
        Object.values(params.filter).some((filter) => !!filter)
      ) {
        handleTaskRunActionInner({
          actionType: params.actionType,
          filter: params.filter,
        });
      }
    },
    [handleTaskRunActionInner],
  );

  return { handleTaskRunAction };
};

type ConfirmActionModalProps = {
  actionType: ActionType;
  params: BaseTaskRunActionParams;
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
  params: BaseTaskRunActionParams;
};

const CancelByExternalIdsContent = ({ label, params }: ModalContentProps) => {
  const { tenantId } = useCurrentTenantId();

  const { data, isLoading, isError } = useQuery({
    ...queries.v1WorkflowRuns.listDisplayNames(
      tenantId,
      params.externalIds || [],
    ),
    enabled: !!params.externalIds,
  });

  if (isLoading || isError) {
    return null;
  }

  const displayNames = data?.rows || [];

  return (
    <div className="flex flex-col gap-y-4">
      <p className="text-md">
        Confirm to {label.toLowerCase()} the following runs:
      </p>
      <ul className="list-disc pl-4 ml-4">
        {displayNames?.slice(0, 10).map((record) => (
          <li className="font-semibold" key={record.metadata.id}>
            {record.displayName}
          </li>
        ))}
        {(displayNames.length || 0) > 10 && (
          <li>{(displayNames.length || 0) - 10} more</li>
        )}
      </ul>
    </div>
  );
};

const ModalContent = ({ label, params }: ModalContentProps) => {
  const { filters, toolbarFilters: tf } = useRunsContext();

  if (params.externalIds?.length) {
    return <CancelByExternalIdsContent label={label} params={params} />;
  } else if (params.filter) {
    const statusToolbarFilter = tf.find((f) => f.columnId === statusKey);
    const additionalMetaToolbarFilter = tf.find(
      (f) => f.columnId === additionalMetadataKey,
    );
    const workflowToolbarFilter = tf.find((f) => f.columnId === workflowKey);

    const hasFilters =
      statusToolbarFilter ||
      additionalMetaToolbarFilter ||
      workflowToolbarFilter;

    return (
      <div className="space-y-6">
        <p className="text-sm text-muted-foreground">
          Confirm to {label.toLowerCase()} all runs matching the following
          filters:
        </p>

        {hasFilters && (
          <div className="space-y-4">
            <h4 className="text-sm font-medium text-foreground">
              Applied Filters
            </h4>
            <div className="space-y-3">
              {statusToolbarFilter && (
                <div className="flex flex-row items-center gap-x-2">
                  <label className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    {statusToolbarFilter.title}
                  </label>
                  <Combobox
                    values={params.filter.statuses}
                    title={statusToolbarFilter.title}
                    type={statusToolbarFilter.type}
                    options={statusToolbarFilter.options}
                    setValues={(values) =>
                      filters.setStatuses(values as V1TaskStatus[])
                    }
                  />
                </div>
              )}
              {additionalMetaToolbarFilter && (
                <div className="gap-x-2 flex flex-row items-center">
                  <label className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    {additionalMetaToolbarFilter.title}
                  </label>
                  <Combobox
                    values={params.filter.additionalMetadata}
                    title={additionalMetaToolbarFilter.title}
                    type={additionalMetaToolbarFilter.type}
                    options={additionalMetaToolbarFilter.options}
                    setValues={(values) => {
                      const kvPairs = values.map((v) => {
                        const [key, value] = v.split(':');
                        return { key, value };
                      });

                      filters.setAllAdditionalMetadata(kvPairs);
                    }}
                  />
                </div>
              )}
              {workflowToolbarFilter && (
                <div className="flex flex-row items-center gap-x-2">
                  <label className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    {workflowToolbarFilter.title}
                  </label>
                  <Combobox
                    values={params.filter.workflowIds}
                    title={workflowToolbarFilter.title}
                    type={workflowToolbarFilter.type}
                    options={workflowToolbarFilter.options}
                    setValues={(values) =>
                      filters.setWorkflowIds(values as string[])
                    }
                  />
                </div>
              )}
              <div className="flex flex-row items-center gap-x-2">
                <label className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Time Range
                </label>
                <TimeFilter className="flex flex-row items-start gap-3 mb-0" />
              </div>
            </div>
          </div>
        )}
      </div>
    );
  } else {
    throw new Error(`Unhandled case: ${params}`);
  }
};

export const ConfirmActionModal = ({
  actionType,
  params,
}: ConfirmActionModalProps) => {
  const label = actionTypeToLabel(actionType);
  const { handleTaskRunAction } = useTaskRunActions();
  const {
    isActionModalOpen,
    actions: { setIsActionModalOpen },
  } = useRunsContext();

  return (
    <Dialog open={isActionModalOpen} onOpenChange={setIsActionModalOpen}>
      <DialogContent className="sm:max-w-[700px] py-8 max-h-screen overflow-auto z-[70]">
        <DialogHeader className="gap-2">
          <div className="flex flex-row justify-between items-center w-full">
            <DialogTitle>{label} runs</DialogTitle>
          </div>
        </DialogHeader>

        <div className="flex flex-col space-y-4">
          <div className="text-sm text-muted-foreground">
            <ModalContent label={label} params={params} />
          </div>

          <div className="flex flex-row items-center gap-3 justify-end pt-4 border-t">
            <Button
              onClick={() => {
                setIsActionModalOpen(false);
              }}
              variant="outline"
            >
              Cancel
            </Button>
            <Button
              onClick={() => {
                handleTaskRunAction({
                  ...params,
                  actionType,
                });
                setIsActionModalOpen(false);
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
  className,
}: {
  disabled: boolean;
  params: TaskRunActionsParams;
  icon: JSX.Element;
  label: string;
  showModal: boolean;
  className?: string;
}) => {
  const { handleTaskRunAction } = useTaskRunActions();
  const {
    actions: {
      setIsActionModalOpen,
      setSelectedActionType,
      setIsActionDropdownOpen,
    },
  } = useRunsContext();

  return (
    <Button
      size={'sm'}
      className={cn('text-sm px-2 py-2 gap-2', className)}
      variant={'outline'}
      disabled={disabled}
      onClick={() => {
        setSelectedActionType(params.actionType);
        setIsActionDropdownOpen(false);

        if (!showModal) {
          handleTaskRunAction(params);
          return;
        }

        setIsActionModalOpen(true);
      }}
    >
      {icon}
      {label}
    </Button>
  );
};

export const TaskRunActionButton = ({
  actionType,
  disabled,
  paramOverrides,
  showModal,
  className,
}: {
  actionType: ActionType;
  disabled: boolean;
  paramOverrides?: BaseTaskRunActionParams;
  showModal: boolean;
  className?: string;
}) => {
  const { actionModalParams } = useRunsContext();
  const params = paramOverrides || actionModalParams;

  switch (actionType) {
    case 'cancel':
      return (
        <BaseActionButton
          disabled={disabled}
          params={{ ...params, actionType: 'cancel' }}
          icon={<XCircleIcon className="w-4 h-4" />}
          label={'Cancel'}
          showModal={showModal}
          className={className}
        />
      );
    case 'replay':
      return (
        <BaseActionButton
          disabled={disabled}
          params={{ ...params, actionType: 'replay' }}
          icon={<Repeat1 className="w-4 h-4" />}
          label={'Replay'}
          showModal={showModal}
          className={className}
        />
      );
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = actionType;
      throw new Error(`Unhandled action type: ${exhaustiveCheck}`);
  }
};
