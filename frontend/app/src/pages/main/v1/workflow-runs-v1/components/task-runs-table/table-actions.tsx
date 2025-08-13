import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { TaskRunActionButton } from '../../../task-runs-v1/actions';
import { useRunsContext } from '../../hooks/runs-provider';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';

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
    display: { showTriggerRunButton, showCancelAndReplayButtons },
  } = useRunsContext();

  return (
    <Popover>
      <PopoverTrigger>
        <Button size="sm">Actions</Button>
      </PopoverTrigger>
      <PopoverContent>
        <div className="flex flex-col items-center gap-y-2 w-full">
          <Button
            key="refresh"
            className="flex flex-row h-8 px-2 lg:px-3 w-full gap-x-2"
            size="sm"
            onClick={onRefresh}
            variant="outline"
            aria-label="Refresh events list"
          >
            <ArrowPathIcon
              className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
            />
            <span>Refetch</span>
          </Button>
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
          {showTriggerRunButton && (
            <Button
              key="trigger"
              className="h-8 border w-full"
              onClick={onTriggerWorkflow}
            >
              Trigger Run
            </Button>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
};
