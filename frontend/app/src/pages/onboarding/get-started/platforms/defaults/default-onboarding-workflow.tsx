import { Button } from '@/components/ui/button';
import api, { WorkflowRun, queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useState } from 'react';

export const DefaultOnboardingWorkflow: React.FC<{
  tenantId: string;
  workerConnected: boolean;
  workflowName?: string;
  setWorkflowTriggered: (val: string) => void;
}> = ({
  tenantId,
  workerConnected,
  workflowName = 'first-workflow',
  setWorkflowTriggered,
}) => {
  const { handleApiError } = useApiError({});

  const listWorkflows = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
    refetchInterval: 5000,
  });

  const workflowId = (listWorkflows.data?.rows ?? []).find(
    (workflow) => workflow.name === workflowName,
  )?.metadata.id;

  const triggerWorkflowMutation = useMutation({
    mutationKey: ['workflow-run:create', workflowId],
    mutationFn: async (input: object) => {
      if (!workflowId) {
        return;
      }

      const res = await api.workflowRunCreate(workflowId, {
        input: input,
      });

      return res.data;
    },
    onSuccess: (workflowRun: WorkflowRun | undefined) => {
      if (!workflowRun) {
        return;
      }

      setWorkflowTriggered(workflowRun.metadata.id);
    },
    onError: handleApiError,
  });

  const [isButtonClicked, setIsButtonClicked] = useState(false);

  const handleButtonClick = () => {
    setIsButtonClicked(true);
    triggerWorkflowMutation.mutate({});
    setTimeout(() => setIsButtonClicked(false), 1000);
  };

  if (!workerConnected) {
    return (
      <div>
        <p>
          Your connection to your worker was lost... please follow instructions
          in the previous step restart your worker
        </p>
      </div>
    );
  }

  return (
    <div>
      <p>Your application is now set up, and your worker is connected!</p>
      <p className="mt-4">
        Click the button below to trigger a run, and check out your worker
        terminal for log output!
      </p>

      <Button
        onClick={handleButtonClick}
        className={`mt-5 ${isButtonClicked ? 'animate-jiggle' : ''}`}
        size="lg"
      >
        Trigger Run
      </Button>
    </div>
  );
};
// TODO
