import { Button } from '@/components/ui/button';
import { useToast } from '@/components/hooks/use-toast';
import api, { V1WorkflowRunDetails, queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

export const DefaultOnboardingWorkflow: React.FC<{
  tenantId: string;
}> = ({ tenantId }) => {
  const workflowName = 'first-workflow';
  const { toast } = useToast();
  const { handleApiError } = useApiError({});
  const navigate = useNavigate();

  const listWorkflows = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
    refetchInterval: 5000,
  });

  const workflow = (listWorkflows.data?.rows ?? []).find(
    (workflow) => workflow.name === workflowName,
  );

  const triggerWorkflowMutation = useMutation({
    mutationKey: ['workflow-run:create', workflow?.metadata.id],
    mutationFn: async (data: { input: object; addlMeta: object }) => {
      if (!workflow) {
        toast({
          title: 'Error',
          description:
            'Workflow not found. Double check that your worker is connected.',
          duration: 5000,
        });

        return;
      }

      const res = await api.v1WorkflowRunCreate(tenantId, {
        workflowName: workflow.name,
        input: data.input,
        additionalMetadata: data.addlMeta,
      });

      return res.data;
    },
    onError: handleApiError,
    onSuccess: (workflowRun: V1WorkflowRunDetails | undefined) => {
      if (!workflowRun) {
        return;
      }

      navigate(`/v1/workflow-runs/${workflowRun.run.metadata.id}`);
    },
  });

  const [isButtonClicked, setIsButtonClicked] = useState(false);

  const handleButtonClick = () => {
    setIsButtonClicked(true);
    triggerWorkflowMutation.mutate({
      input: {},
      addlMeta: {},
    });
    setTimeout(() => setIsButtonClicked(false), 1000);
  };

  return (
    <div>
      <p className="mt-4 text-muted-foreground">
        Your application is now set up, and your worker is connected!
      </p>
      <p className="mt-4 text-muted-foreground">
        Click the button below to trigger a run, and check out your worker
        terminal for log output.
      </p>

      <Button
        onClick={handleButtonClick}
        className={`mt-5 ${isButtonClicked ? 'animate-jiggle' : ''}`}
        variant={'outline'}
      >
        Trigger Run
      </Button>
    </div>
  );
};
// TODO
