import { ReviewedButtonTemp } from '@/components/v1/ui/button';
import { TaskRunActionButton } from '../../../task-runs-v1/actions';
import { useRunsContext } from '../../hooks/runs-provider';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { useMemo, useState } from 'react';
import { Play, Command } from 'lucide-react';
import { ChevronDownIcon } from '@radix-ui/react-icons';

interface TableActionsProps {
  onTriggerWorkflow: () => void;
}

export const TableActions = ({ onTriggerWorkflow }: TableActionsProps) => {
  const [shouldDelayClose, setShouldDelayClose] = useState(false);
  const {
    isActionDropdownOpen,
    actions: { setIsActionDropdownOpen },
    display: { hideTriggerRunButton, hideCancelAndReplayButtons },
  } = useRunsContext();

  const actions = useMemo(() => {
    let baseActions = [
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
            <ReviewedButtonTemp variant="outline" size="sm">
              <Command className="h-4 w-4 cq-xl:hidden" />
              <span className="cq-xl:inline hidden text-sm">Actions</span>
              <ChevronDownIcon className="h-4 w-4 ml-2 hidden cq-xl:inline" />
            </ReviewedButtonTemp>
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
        <ReviewedButtonTemp key="trigger" size="sm" onClick={onTriggerWorkflow}>
          <span className="cq-xl:inline hidden text-sm">Trigger Run</span>
          {/* important: this icon can't be the `rightIcon` in the button b/c it's dynamically shown */}
          <Play className="size-4 cq-xl:hidden" />
        </ReviewedButtonTemp>,
        ...baseActions,
      ];
    }

    return baseActions;
  }, [
    onTriggerWorkflow,
    hideTriggerRunButton,
    hideCancelAndReplayButtons,
    setIsActionDropdownOpen,
    isActionDropdownOpen,
    shouldDelayClose,
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
