import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { TaskRunActionButton } from '../../../task-runs-v1/actions';
import { useRunsContext } from '../../hooks/runs-provider';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import { useMemo } from 'react';
import { Snowflake } from 'lucide-react';

interface TableActionsProps {
  taskIdsPendingAction: string[];
  onRefresh: () => void;
  onActionProcessed: (action: 'cancel' | 'replay', ids: string[]) => void;
  onTriggerWorkflow: () => void;
  rotate: boolean;
  toast: any;
}

export const TableActions = ({
  taskIdsPendingAction,
  onRefresh,
  onActionProcessed,
  onTriggerWorkflow,
  rotate,
  toast,
}: TableActionsProps) => {
  const {
    state: { hasRowsSelected, hasFiltersApplied },
    selectedRuns,
    filters,
    isFrozen,
    actions: { setIsFrozen },
    display: { showTriggerRunButton, showCancelAndReplayButtons },
  } = useRunsContext();

  const actions = useMemo(() => {
    let baseActions = [
      <Button
        key="refresh"
        className="h-8 px-2 lg:px-3"
        size="sm"
        onClick={onRefresh}
        variant="outline"
        aria-label="Refresh events list"
      >
        <ArrowPathIcon
          className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
        />
      </Button>,
      <Button
        key="freeze"
        className="h-8 px-2 lg:px-3"
        size="sm"
        onClick={() => setIsFrozen(!isFrozen)}
        variant={isFrozen ? 'default' : 'outline'}
        aria-label="Refresh events list"
      >
        <Snowflake
          className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
        />
      </Button>,
    ];

    if (showCancelAndReplayButtons) {
      baseActions = [
        <CancelReplayActions
          key="cancel-replay"
          taskIdsPendingAction={taskIdsPendingAction}
          onActionProcessed={onActionProcessed}
          toast={toast}
        />,
        ...baseActions,
      ];
    }

    if (showTriggerRunButton) {
      baseActions = [
        <Button
          key="trigger"
          className="h-8 border"
          onClick={onTriggerWorkflow}
        >
          Trigger Run
        </Button>,
        ...baseActions,
      ];
    }

    return baseActions;
  }, [
    hasRowsSelected,
    hasFiltersApplied,
    selectedRuns,
    taskIdsPendingAction.length,
    onRefresh,
    onActionProcessed,
    onTriggerWorkflow,
    showTriggerRunButton,
    rotate,
    toast,
    filters,
    showCancelAndReplayButtons,
  ]);

  return <>{actions}</>;
};

export const CancelReplayActions = ({
  taskIdsPendingAction,
  onActionProcessed,
  toast,
}: Pick<
  TableActionsProps,
  'taskIdsPendingAction' | 'toast' | 'onActionProcessed'
>) => {
  const {
    state: { hasRowsSelected, hasFiltersApplied },
    selectedRuns,
    filters,
    display: { showCancelAndReplayButtons },
  } = useRunsContext();

  return (
    <Popover>
      <PopoverTrigger>
        <Button size="sm">Actions</Button>
      </PopoverTrigger>
      <PopoverContent>
        <div className="flex flex-col items-center gap-y-2 w-full">
          {showCancelAndReplayButtons && (
            <TaskRunActionButton
              key="cancel"
              actionType="cancel"
              disabled={
                !(hasRowsSelected || hasFiltersApplied) ||
                taskIdsPendingAction.length > 0
              }
              params={
                selectedRuns.length > 0
                  ? { externalIds: selectedRuns.map((run) => run?.metadata.id) }
                  : {
                      filter: {
                        ...filters.apiFilters,
                        since: filters.apiFilters.since || '',
                      },
                    }
              }
              showModal
              onActionProcessed={(ids) => onActionProcessed('cancel', ids)}
              onActionSubmit={() => {
                toast({
                  title: 'Cancel request submitted',
                  description: "No need to hit 'Cancel' again.",
                });
              }}
              className="w-full"
            />
          )}
          {showCancelAndReplayButtons && (
            <TaskRunActionButton
              key="replay"
              actionType="replay"
              disabled={
                !(hasRowsSelected || hasFiltersApplied) ||
                taskIdsPendingAction.length > 0
              }
              params={
                selectedRuns.length > 0
                  ? { externalIds: selectedRuns.map((run) => run?.metadata.id) }
                  : {
                      filter: {
                        ...filters.apiFilters,
                        since: filters.apiFilters.since || '',
                      },
                    }
              }
              showModal
              onActionProcessed={(ids) => onActionProcessed('replay', ids)}
              onActionSubmit={() => {
                toast({
                  title: 'Replay request submitted',
                  description: "No need to hit 'Replay' again.",
                });
              }}
              className="w-full"
            />
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
};
