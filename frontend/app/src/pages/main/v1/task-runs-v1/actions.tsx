import { Button } from '@/components/v1/ui/button';
import {
  DialogTitle,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
} from '@/components/v1/ui/dialog';
import api, {
  queries,
  V1CancelTaskRequest,
  V1ReplayTaskRequest,
  V1TaskStatus,
} from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { ArrowPathIcon, XCircleIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useCallback, useState } from 'react';
import {
  TimeWindow,
  useColumnFilters,
} from '../workflow-runs-v1/hooks/column-filters';
import { useToolbarFilters } from '../workflow-runs-v1/hooks/toolbar-filters';
import { Combobox } from '@/components/v1/molecules/combobox/combobox';
import { TaskRunColumn } from '../workflow-runs-v1/components/v1/task-runs-columns';
import {
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
  Select,
} from '@/components/v1/ui/select';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { useCurrentTenantId } from '@/hooks/use-tenant';

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
  const { tenantId } = useCurrentTenantId();

  const { handleApiError } = useApiError({});

  const { mutate: handleAction } = useMutation({
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
  const tf = useToolbarFilters({
    filterVisibility: {},
  });
  const cf = useColumnFilters();

  if (params.externalIds?.length) {
    return <CancelByExternalIdsContent label={label} params={params} />;
  } else if (params.filter) {
    const statusToolbarFilter = tf.find(
      (f) => f.columnId === TaskRunColumn.status,
    );
    const additionalMetaToolbarFilter = tf.find(
      (f) => f.columnId === TaskRunColumn.additionalMetadata,
    );
    const workflowToolbarFilter = tf.find(
      (f) => f.columnId === TaskRunColumn.workflow,
    );

    return (
      <div className="gap-y-4 flex flex-col">
        <p className="text-md">
          Confirm to {label.toLowerCase()} all runs matching the following
          filters:
        </p>
        <div className="grid grid-cols-2 gap-x-2 items-start justify-start gap-y-4">
          {statusToolbarFilter && (
            <Combobox
              values={params.filter.statuses}
              title={statusToolbarFilter.title}
              type={statusToolbarFilter.type}
              options={statusToolbarFilter.options}
              setValues={(values) => cf.setStatus(values[0] as V1TaskStatus)}
            />
          )}
          {additionalMetaToolbarFilter && (
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

                cf.setAllAdditionalMetadata({ kvPairs });
              }}
            />
          )}
          {workflowToolbarFilter && (
            <Combobox
              values={params.filter.workflowIds}
              title={workflowToolbarFilter.title}
              type={workflowToolbarFilter.type}
              options={workflowToolbarFilter.options}
              setValues={(values) => cf.setWorkflowId(values[0] as string)}
            />
          )}
          <Select
            value={
              cf.filters.isCustomTimeRange ? 'custom' : cf.filters.timeWindow
            }
            onValueChange={(value: TimeWindow | 'custom') => {
              if (value !== 'custom') {
                cf.setFilterValues([
                  { key: 'isCustomTimeRange', value: false },
                  { key: 'timeWindow', value: value },
                ]);
              } else {
                cf.setFilterValues([{ key: 'isCustomTimeRange', value: true }]);
              }
            }}
          >
            <SelectTrigger className="flex flex-1 h-8">
              <SelectValue id="timerange" placeholder="Choose time range" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1h">1 hour</SelectItem>
              <SelectItem value="6h">6 hours</SelectItem>
              <SelectItem value="1d">1 day</SelectItem>
              <SelectItem value="7d">7 days</SelectItem>
              <SelectItem value="custom">Custom</SelectItem>
            </SelectContent>
          </Select>
        </div>
        {cf.filters.isCustomTimeRange && (
          <div className="flex flex-row w-full flex-1 gap-x-2 items-start justify-start gap-y-4">
            <DateTimePicker
              key="after"
              label="After"
              date={
                cf.filters.createdAfter
                  ? new Date(cf.filters.createdAfter)
                  : undefined
              }
              setDate={(date) => {
                cf.setCreatedAfter(date?.toISOString());
              }}
            />
            <DateTimePicker
              key="before"
              label="Before"
              date={
                cf.filters.finishedBefore
                  ? new Date(cf.filters.finishedBefore)
                  : undefined
              }
              setDate={(date) => {
                cf.setFinishedBefore(date?.toISOString());
              }}
            />
            <Button
              key="clear"
              onClick={() => {
                cf.setCustomTimeRange(undefined);
              }}
              variant="outline"
              size="sm"
              className="text-xs h-9 py-2 flex-1"
            >
              <XCircleIcon className="h-[18px] w-[18px] mr-2" />
              Clear
            </Button>{' '}
          </div>
        )}
      </div>
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
      <DialogContent className="sm:max-w-[800px] py-12 max-h-screen overflow-auto">
        <DialogHeader className="gap-2">
          <div className="flex flex-row justify-between items-center w-full">
            <DialogTitle>{label} runs</DialogTitle>
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
