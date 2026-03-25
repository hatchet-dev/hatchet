import { useRunsContext } from '../workflow-runs-v1/hooks/runs-provider';
import { useToast } from '@/components/v1/hooks/use-toast';
import { IDGetter } from '@/components/v1/molecules/data-table/data-table';
import {
  DataTableOptionsContent,
  DataTableOptionsContentProps,
} from '@/components/v1/molecules/data-table/data-table-options';
import { Button } from '@/components/v1/ui/button';
import {
  DialogTitle,
  Dialog,
  DialogContent,
  DialogHeader,
} from '@/components/v1/ui/dialog';
import {
  PortalTooltip,
  PortalTooltipContent,
  PortalTooltipProvider,
  PortalTooltipTrigger,
} from '@/components/v1/ui/portal-tooltip';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, {
  queries,
  V1CancelTaskRequest,
  V1ReplayTaskRequest,
} from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { cn } from '@/lib/utils';
import { XCircleIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery } from '@tanstack/react-query';
import { capitalize } from 'lodash';
import { Repeat1 } from 'lucide-react';
import { useCallback } from 'react';

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

const useTaskRunActions = () => {
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
      <ul className="ml-4 list-disc pl-4">
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

function ModalContent<TData extends IDGetter<TData>>({
  label,
  params,
  table,
  columnKeyToName,
  filters,
  hiddenFilters,
}: ModalContentProps & DataTableOptionsContentProps<TData>) {
  if (params.externalIds?.length) {
    return <CancelByExternalIdsContent label={label} params={params} />;
  } else if (params.filter) {
    return (
      <div className="space-y-6">
        <p className="text-sm text-muted-foreground">
          Confirm to {label.toLowerCase()} all runs matching the following
          filters:
        </p>
        <DataTableOptionsContent
          table={table}
          filters={filters}
          columnKeyToName={columnKeyToName}
          hiddenFilters={hiddenFilters}
          showColumnVisibility={false}
        />
      </div>
    );
  } else {
    throw new Error(`Unhandled case: ${params}`);
  }
}

export function ConfirmActionModal<TData extends IDGetter<TData>>({
  actionType,
  params,
  table,
  columnKeyToName,
  filters,
  hiddenFilters,
}: ConfirmActionModalProps & DataTableOptionsContentProps<TData>) {
  const label = actionTypeToLabel(actionType);
  const { handleTaskRunAction } = useTaskRunActions();
  const {
    isActionModalOpen,
    actions: { setIsActionModalOpen },
  } = useRunsContext();

  return (
    <Dialog open={isActionModalOpen} onOpenChange={setIsActionModalOpen}>
      <DialogContent className="max-h-[90%] overflow-auto py-8 sm:max-w-[700px]">
        <DialogHeader className="gap-2">
          <div className="flex w-full flex-row items-center justify-between">
            <DialogTitle>{label} runs</DialogTitle>
          </div>
        </DialogHeader>

        <div className="flex flex-col space-y-4">
          <div className="text-sm text-muted-foreground">
            <ModalContent
              label={label}
              params={params}
              table={table}
              filters={filters}
              columnKeyToName={columnKeyToName}
              hiddenFilters={hiddenFilters}
              showColumnVisibility={false}
            />
          </div>

          <div className="flex flex-row items-center justify-end gap-3 border-t pt-4">
            <Button
              onClick={() => {
                setIsActionModalOpen(false);
              }}
              variant="outline"
            >
              Close
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
}

const BaseActionButton = ({
  disabled,
  params,
  icon,
  label,
  showModal,
  showLabel,
  className,
}: {
  disabled: boolean;
  params: TaskRunActionsParams;
  icon: JSX.Element;
  label: string;
  showLabel: boolean;
  showModal: boolean;
  className?: string;
}) => {
  const { handleTaskRunAction } = useTaskRunActions();
  const {
    actions: { setIsActionModalOpen, setSelectedActionType },
  } = useRunsContext();

  return (
    <PortalTooltipProvider>
      <PortalTooltip>
        <PortalTooltipTrigger>
          <Button
            size={'sm'}
            className={cn('text-sm gap-2', className)}
            variant={'outline'}
            disabled={disabled}
            onClick={() => {
              setSelectedActionType(params.actionType);

              if (!showModal) {
                handleTaskRunAction(params);
                return;
              }

              setIsActionModalOpen(true);
            }}
          >
            {icon}
            {showLabel && label}
          </Button>
        </PortalTooltipTrigger>
        <PortalTooltipContent>{label}</PortalTooltipContent>
      </PortalTooltip>
    </PortalTooltipProvider>
  );
};

export const TaskRunActionButton = ({
  actionType,
  disabled,
  paramOverrides,
  showModal,
  showLabel,
  className,
}: {
  actionType: ActionType;
  disabled: boolean;
  paramOverrides?: BaseTaskRunActionParams;
  showModal: boolean;
  showLabel: boolean;
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
          icon={<XCircleIcon className="size-4" />}
          label={'Cancel'}
          showModal={showModal}
          className={className}
          showLabel={showLabel}
        />
      );
    case 'replay':
      return (
        <BaseActionButton
          disabled={disabled}
          params={{ ...params, actionType: 'replay' }}
          icon={<Repeat1 className="size-4" />}
          label={'Replay'}
          showModal={showModal}
          className={className}
          showLabel={showLabel}
        />
      );
    default:
      const exhaustiveCheck: never = actionType;
      throw new Error(`Unhandled action type: ${exhaustiveCheck}`);
  }
};
