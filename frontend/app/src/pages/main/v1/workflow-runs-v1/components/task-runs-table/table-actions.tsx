import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { TaskRunActionButton } from '../../../task-runs-v1/actions';
import { useRunsContext } from '../../hooks/runs-provider';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { useMemo, useState } from 'react';
import { Snowflake } from 'lucide-react';
import { ChevronDownIcon } from '@radix-ui/react-icons';

interface TableActionsProps {
  onRefresh: () => void;
  onActionProcessed: (action: 'cancel' | 'replay', ids: string[]) => void;
  onTriggerWorkflow: () => void;
  rotate: boolean;
  toast: any;
}

export const TableActions = ({
  onRefresh,
  onActionProcessed,
  onTriggerWorkflow,
  rotate,
  toast,
}: TableActionsProps) => {
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const [shouldDelayClose, setShouldDelayClose] = useState(false);
  const {
    isFrozen,
    actions: { setIsFrozen },
    display: { hideTriggerRunButton, hideCancelAndReplayButtons },
    isActionModalOpen,
  } = useRunsContext();

  const actions = useMemo(() => {
    let baseActions = [
      <DropdownMenu
        key="actions"
        open={dropdownOpen}
        onOpenChange={(open) => {
          if (open) {
            setDropdownOpen(true);
            setShouldDelayClose(false);
          } else if (shouldDelayClose) {
            setTimeout(() => setDropdownOpen(false), 150);
            setShouldDelayClose(false);
          } else {
            setDropdownOpen(false);
          }
        }}
      >
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="h-8">
            Actions
            <ChevronDownIcon className="ml-2 h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="end"
          className="z-[70] data-[is-modal-open=true]:hidden"
          data-is-modal-open={isActionModalOpen}
        >
          {!hideCancelAndReplayButtons && (
            <>
              <CancelMenuItem
                onActionProcessed={(action, ids) => {
                  onActionProcessed(action, ids);
                  setDropdownOpen(false);
                }}
                toast={toast}
                onDelayedClose={() => setShouldDelayClose(true)}
              />
              <ReplayMenuItem
                onActionProcessed={(action, ids) => {
                  onActionProcessed(action, ids);
                  setDropdownOpen(false);
                }}
                toast={toast}
                onDelayedClose={() => setShouldDelayClose(true)}
              />
              <DropdownMenuSeparator />
            </>
          )}
          <DropdownMenuItem
            onClick={() => {
              setShouldDelayClose(true);
              onRefresh();
            }}
          >
            <ArrowPathIcon
              className={`mr-2 h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
            />
            Refresh
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => {
              setShouldDelayClose(true);
              setIsFrozen(!isFrozen);
            }}
            className="text-sm"
          >
            <Snowflake className="mr-2 h-4 w-4" />
            {isFrozen ? 'Unfreeze' : 'Freeze'}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>,
    ];

    if (!hideTriggerRunButton) {
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
    onRefresh,
    onActionProcessed,
    onTriggerWorkflow,
    hideTriggerRunButton,
    rotate,
    toast,
    hideCancelAndReplayButtons,
    isFrozen,
    setIsFrozen,
    dropdownOpen,
    shouldDelayClose,
    isActionModalOpen,
  ]);

  return <>{actions}</>;
};

const CancelMenuItem = ({
  onActionProcessed,
  toast,
  onDelayedClose,
}: Pick<TableActionsProps, 'toast' | 'onActionProcessed'> & {
  onDelayedClose: () => void;
}) => {
  const { selectedRuns, filters } = useRunsContext();

  const params =
    selectedRuns.length > 0
      ? { externalIds: selectedRuns.map((run) => run?.metadata.id) }
      : {
          filter: {
            ...filters.apiFilters,
            since: filters.apiFilters.since || '',
          },
        };

  return (
    <div className="w-full">
      <TaskRunActionButton
        actionType="cancel"
        disabled={false}
        params={params}
        showModal
        onActionProcessed={(ids) => onActionProcessed('cancel', ids)}
        onActionSubmit={() => {
          onDelayedClose();
          toast({
            title: 'Cancel request submitted',
            description: "No need to hit 'Cancel' again.",
          });
        }}
        className="w-full justify-start h-8 px-2 py-1.5 font-normal border-0 bg-transparent hover:bg-accent hover:text-accent-foreground rounded-sm"
      />
    </div>
  );
};

const ReplayMenuItem = ({
  onActionProcessed,
  toast,
  onDelayedClose,
}: Pick<TableActionsProps, 'toast' | 'onActionProcessed'> & {
  onDelayedClose: () => void;
}) => {
  const { selectedRuns, filters } = useRunsContext();

  const params =
    selectedRuns.length > 0
      ? { externalIds: selectedRuns.map((run) => run?.metadata.id) }
      : {
          filter: {
            ...filters.apiFilters,
            since: filters.apiFilters.since || '',
          },
        };

  return (
    <div className="w-full">
      <TaskRunActionButton
        actionType="replay"
        disabled={false}
        params={params}
        showModal
        onActionProcessed={(ids) => onActionProcessed('replay', ids)}
        onActionSubmit={() => {
          onDelayedClose();
          toast({
            title: 'Replay request submitted',
            description: "No need to hit 'Replay' again.",
          });
        }}
        className="w-full justify-start h-8 px-2 py-1.5 font-normal border-0 bg-transparent hover:bg-accent hover:text-accent-foreground rounded-sm"
      />
    </div>
  );
};
