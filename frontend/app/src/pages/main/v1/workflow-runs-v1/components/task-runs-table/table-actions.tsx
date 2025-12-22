import { TaskRunActionButton } from '../../../task-runs-v1/actions';
import { useRunsContext } from '../../hooks/runs-provider';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { ChevronDownIcon } from '@radix-ui/react-icons';
import { Play, Command } from 'lucide-react';
import { useMemo, useState } from 'react';

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
            <Button variant="outline" size="sm">
              <Command className="cq-xl:hidden size-4" />
              <span className="cq-xl:inline hidden text-sm">Actions</span>
              <ChevronDownIcon className="cq-xl:inline ml-2 hidden size-4" />
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
        <Button key="trigger" size="sm" onClick={onTriggerWorkflow}>
          <span className="cq-xl:inline hidden text-sm">Trigger Run</span>
          {/* important: this icon can't be the `rightIcon` in the button b/c it's dynamically shown */}
          <Play className="cq-xl:hidden size-4" />
        </Button>,
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
        className="h-8 w-full justify-start rounded-sm border-0 bg-transparent px-2 py-1.5 font-normal hover:bg-accent hover:text-accent-foreground"
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
        className="h-8 w-full justify-start rounded-sm border-0 bg-transparent px-2 py-1.5 font-normal hover:bg-accent hover:text-accent-foreground"
      />
    </div>
  );
};
