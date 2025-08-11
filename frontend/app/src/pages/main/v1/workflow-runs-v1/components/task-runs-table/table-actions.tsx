import { useMemo } from 'react';
import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { TaskRunActionButton } from '../../../task-runs-v1/actions';
import { V1TaskStatus } from '@/lib/api';
import { FilterActions, APIFilters } from '../../hooks/use-runs-table-filters';

interface TableActionsProps {
  hasRowsSelected: boolean;
  hasFiltersApplied: boolean;
  selectedRuns: any[];
  apiFilters: {
    since?: string;
    until?: string;
    statuses?: V1TaskStatus[];
    workflowIds?: string[];
    additionalMetadata?: string[];
  };
  taskIdsPendingAction: string[];
  onRefresh: () => void;
  onActionProcessed: (action: 'cancel' | 'replay', ids: string[]) => void;
  onTriggerWorkflow: () => void;
  showTriggerRunButton: boolean;
  rotate: boolean;
  toast: any;
  filters: FilterActions & { apiFilters: APIFilters };
}

export const TableActions = ({
  hasRowsSelected,
  hasFiltersApplied,
  selectedRuns,
  apiFilters,
  taskIdsPendingAction,
  onRefresh,
  onActionProcessed,
  onTriggerWorkflow,
  showTriggerRunButton,
  rotate,
  toast,
  filters,
}: TableActionsProps) => {
  const actions = useMemo(() => {
    const baseActions = [
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
            : { filter: { ...apiFilters, since: apiFilters.since || '' } }
        }
        showModal
        onActionProcessed={(ids) => onActionProcessed('cancel', ids)}
        onActionSubmit={() => {
          toast({
            title: 'Cancel request submitted',
            description: "No need to hit 'Cancel' again.",
          });
        }}
        filters={filters}
      />,
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
            : { filter: { ...apiFilters, since: apiFilters.since || '' } }
        }
        showModal
        onActionProcessed={(ids) => onActionProcessed('replay', ids)}
        onActionSubmit={() => {
          toast({
            title: 'Replay request submitted',
            description: "No need to hit 'Replay' again.",
          });
        }}
        filters={filters}
      />,
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
    ];

    if (showTriggerRunButton) {
      return [
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
    apiFilters,
    taskIdsPendingAction.length,
    onRefresh,
    onActionProcessed,
    onTriggerWorkflow,
    showTriggerRunButton,
    rotate,
    toast,
  ]);

  return <>{actions}</>;
};
