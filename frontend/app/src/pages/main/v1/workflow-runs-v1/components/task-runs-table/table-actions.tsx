import { TaskRunActionButton } from '../../../task-runs-v1/actions';
import { useRunsContext } from '../../hooks/runs-provider';
import { Button } from '@/components/v1/ui/button';
import { Play } from 'lucide-react';
import { useMemo } from 'react';

interface TableActionsProps {
  onTriggerWorkflow: () => void;
}

export const TableActions = ({ onTriggerWorkflow }: TableActionsProps) => {
  const {
    display: { hideTriggerRunButton, hideCancelAndReplayButtons },
  } = useRunsContext();

  const actions = useMemo(() => {
    let baseActions = [
      !hideCancelAndReplayButtons && (
        <div className="flex flex-row gap-x-1">
          <TaskRunActionButton
            actionType="cancel"
            disabled={false}
            showModal
            showLabel={false}
          />
          <TaskRunActionButton
            actionType="replay"
            disabled={false}
            showModal
            showLabel={false}
          />
        </div>
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
  }, [onTriggerWorkflow, hideTriggerRunButton, hideCancelAndReplayButtons]);

  return <>{actions}</>;
};
