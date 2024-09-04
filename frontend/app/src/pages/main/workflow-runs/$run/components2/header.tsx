import { WorkflowRunState } from '@/lib/api';
import React, { useMemo } from 'react';
import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/button';

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { BiDotsVertical } from 'react-icons/bi';
import { Dialog } from '@/components/ui/dialog';
import { RunStatus } from '../../components/run-statuses';

interface RunDetailHeaderProps {
  data?: WorkflowRunState;
  loading?: boolean;
}

const RunDetailHeader: React.FC<RunDetailHeaderProps> = ({ data, loading }) => {
  const [displayName, runId] = useMemo(() => {
    const parts = data?.displayName?.split('-');

    console.log('parts', parts);
    if (!parts) {
      return [null, null];
    }

    return [parts[0], parts[1]];
  }, [data?.displayName]);

  if (loading || !data) {
    return <div>Loading...</div>;
  }

  return (
    <div className="flex flex-row justify-between items-center">
      <div>
        <h2 className="text-2xl font-bold leading-tight text-foreground flex flex-row  items-center">
          <Link to={`/workflows/${data?.workflowVersionId}`}>
            {displayName}
          </Link>
          /{runId || data.metadata.id}
          {/* /{selectedStepRun?.step?.readableId || '*'} */}
        </h2>
      </div>
      <div className="flex flex-row gap-2 items-center">
        <RunStatus status={data.status} className="text-sm mt-1 px-4 shrink" />

        <DropdownMenu>
          <DropdownMenuTrigger>
            <Button aria-label="Workflow Actions" size="icon" variant="outline">
              <BiDotsVertical />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              onClick={() => {
                // setShowInputDialog(true);
              }}
            >
              View workflow input
            </DropdownMenuItem>
            <DropdownMenuItem
            //   disabled={!WORKFLOW_RUN_TERMINAL_STATUSES.includes(run.status)}
            //   onClick={() => {
            //     replayWorkflowRunsMutation.mutate();
            //   }}
            >
              Replay workflow
            </DropdownMenuItem>
            <DropdownMenuItem
              //   disabled={WORKFLOW_RUN_TERMINAL_STATUSES.includes(run.status)}
              onClick={() => {
                // cancelWorkflowRunMutation.mutate();
              }}
            >
              Cancel all running steps
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <Dialog
        // open={!!showInputDialog}
        onOpenChange={(open) => {
          if (!open) {
            // setShowInputDialog(false);
          }
        }}
      >
        {/* {showInputDialog && <WorkflowRunInputDialog wr={run} />} */}
      </Dialog>
    </div>
  );
};

export default RunDetailHeader;
