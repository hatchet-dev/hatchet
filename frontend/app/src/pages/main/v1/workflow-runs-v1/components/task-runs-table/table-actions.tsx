import { Button } from '@/components/v1/ui/button';
import { TaskRunActionButton } from '../../../task-runs-v1/actions';
import { useRunsContext } from '../../hooks/runs-provider';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { useMemo, useState } from 'react';
import { Play, MoreHorizontal } from 'lucide-react';
import { ChevronDownIcon } from '@radix-ui/react-icons';
import { RefetchIntervalDropdown } from '@/components/refetch-interval-dropdown';

interface TableActionsProps {
  onRefresh: () => void;
  onTriggerWorkflow: () => void;
}

export const TableActions = ({
  onRefresh,
  onTriggerWorkflow,
}: TableActionsProps) => {
  const [shouldDelayClose, setShouldDelayClose] = useState(false);
  const {
    isActionDropdownOpen,
    actions: { setIsActionDropdownOpen, refetchRuns, refetchMetrics },
    display: { hideTriggerRunButton, hideCancelAndReplayButtons },
    isRefetching,
  } = useRunsContext();

  const actions = useMemo(() => {
    let baseActions = [
      <RefetchIntervalDropdown
        key="refetch-interval"
        isRefetching={isRefetching}
        onRefetch={() => {
          onRefresh();
          refetchRuns();
          refetchMetrics();
        }}
      />,
      !hideCancelAndReplayButtons && (
        <DropdownMenu
          key="actions"
          open={isActionDropdownOpen}
          onOpenChange={(open) => {
            if (open) {
              setIsActionDropdownOpen(true);
              setShouldDelayClose(false);
            } else if (shouldDelayClose) {
              setTimeout(() => setIsActionDropdownOpen(false), 150);
              setShouldDelayClose(false);
            } else {
              setIsActionDropdownOpen(false);
            }
          }}
        >
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="h-8">
              <MoreHorizontal className="h-4 w-4 cq-xl:hidden" />
              <span className="cq-xl:inline hidden">Actions</span>
              <ChevronDownIcon className="h-4 w-4 ml-2 hidden cq-xl:inline" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="z-[70]">
            <CancelMenuItem />
            <ReplayMenuItem />
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    ];

    if (!hideTriggerRunButton) {
      baseActions = [
        <Button
          key="trigger"
          className="h-8 border ml-2"
          onClick={onTriggerWorkflow}
        >
          <span className="cq-xl:inline hidden">Trigger Run</span>
          <Play className="size-4 cq-xl:hidden" />
        </Button>,
        ...baseActions,
      ];
    }

    return baseActions;
  }, [
    onRefresh,
    onTriggerWorkflow,
    hideTriggerRunButton,
    hideCancelAndReplayButtons,
    setIsActionDropdownOpen,
    isActionDropdownOpen,
    shouldDelayClose,
    refetchRuns,
    refetchMetrics,
    isRefetching,
  ]);

  return <>{actions}</>;
};

const CancelMenuItem = () => {
  return (
    <div className="w-full">
      <TaskRunActionButton
        actionType="cancel"
        disabled={false}
        showModal
        className="w-full justify-start h-8 px-2 py-1.5 font-normal border-0 bg-transparent hover:bg-accent hover:text-accent-foreground rounded-sm"
      />
    </div>
  );
};

const ReplayMenuItem = () => {
  return (
    <div className="w-full">
      <TaskRunActionButton
        actionType="replay"
        disabled={false}
        showModal
        className="w-full justify-start h-8 px-2 py-1.5 font-normal border-0 bg-transparent hover:bg-accent hover:text-accent-foreground rounded-sm"
      />
    </div>
  );
};
