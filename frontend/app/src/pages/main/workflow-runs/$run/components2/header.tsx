import React, { useMemo } from 'react';
import { Link, useOutletContext } from 'react-router-dom';

import { Dialog } from '@/components/ui/dialog';
import { RunStatus } from '../../components/run-statuses';
import api, { WorkflowRunShape, WorkflowRunStatus } from '@/lib/api';
import { useMutation } from '@tanstack/react-query';
import invariant from 'tiny-invariant';
import { useApiError } from '@/lib/hooks';
import { TenantContextType } from '@/lib/outlet';
import { Button } from '@/components/ui/button';

interface RunDetailHeaderProps {
  data?: WorkflowRunShape;
  loading?: boolean;
  refetch: () => void;
}

export const WORKFLOW_RUN_TERMINAL_STATUSES = [
  WorkflowRunStatus.CANCELLED,
  WorkflowRunStatus.FAILED,
  WorkflowRunStatus.SUCCEEDED,
];

const RunDetailHeader: React.FC<RunDetailHeaderProps> = ({
  data,
  loading,
  refetch,
}) => {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [displayName, runId] = useMemo(() => {
    const parts = data?.displayName?.split('-');
    if (!parts) {
      return [null, null];
    }

    return [parts[0], parts[1]];
  }, [data?.displayName]);

  const { handleApiError } = useApiError({});

  const cancelWorkflowRunMutation = useMutation({
    mutationKey: ['workflow-run:cancel', data?.tenantId, data?.metadata.id],
    mutationFn: async () => {
      const tenantId = data?.tenantId;
      const workflowRunId = data?.metadata.id;

      invariant(tenantId, 'has tenantId');
      invariant(workflowRunId, 'has tenantId');

      const res = await api.workflowRunCancel(tenantId, {
        workflowRunIds: [workflowRunId],
      });

      return res.data;
    },
    onError: handleApiError,
  });

  const replayWorkflowRunsMutation = useMutation({
    mutationKey: ['workflow-run:update:replay', tenant.metadata.id],
    mutationFn: async () => {
      if (!data) {
        return;
      }

      await api.workflowRunUpdateReplay(tenant.metadata.id, {
        workflowRunIds: [data?.metadata.id],
      });
    },
    onSuccess: () => {
      refetch();
    },
    onError: handleApiError,
  });

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
        <Button
          disabled={!WORKFLOW_RUN_TERMINAL_STATUSES.includes(data.status)}
          onClick={() => {
            replayWorkflowRunsMutation.mutate();
          }}
        >
          Replay workflow
        </Button>
        <Button
          disabled={WORKFLOW_RUN_TERMINAL_STATUSES.includes(data.status)}
          onClick={() => {
            cancelWorkflowRunMutation.mutate();
          }}
        >
          Cancel all running steps
        </Button>
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
