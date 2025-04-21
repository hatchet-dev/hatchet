import { useApiError, useApiMetaIntegrations } from '@/lib/hooks';
import api, { queries, WorkflowUpdateRequest } from '@/next/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';

export const useWorkflowDetails = ({ workflowId }: { workflowId: string }) => {
  const { handleApiError } = useApiError({});

  const workflowQuery = useQuery({
    ...queries.workflows.get(workflowId),
    refetchInterval: 1000,
  });

  const workflowVersionQuery = useQuery({
    ...queries.workflows.getVersion(workflowId),
    refetchInterval: 1000,
  });

  const navigate = useNavigate();

  const updateWorkflowMutation = useMutation({
    mutationKey: ['workflow:update', workflowQuery?.data?.metadata.id],
    mutationFn: async (data: WorkflowUpdateRequest) => {
      const res = await api.workflowUpdate(workflowId, {
        ...data,
      });

      return res.data;
    },
    onError: handleApiError,
    onSuccess: () => {
      workflowQuery.refetch();
    },
  });

  const pauseWorkflow = () => {
    updateWorkflowMutation.mutate({ isPaused: true });
  };

  const unpauseWorkflow = () => {
    updateWorkflowMutation.mutate({ isPaused: false });
  };

  const deleteWorkflowMutation = useMutation({
    mutationKey: ['workflow:delete', workflowQuery?.data?.metadata.id],
    mutationFn: async () => {
      if (!workflowQuery?.data) {
        return;
      }

      const res = await api.workflowDelete(workflowQuery?.data.metadata.id);

      return res.data;
    },
    onSuccess: () => {
      navigate('/v1/next/runs');
    },
  });

  const deleteWorkflow = () => {
    deleteWorkflowMutation.mutate();
  };

  const integrations = useApiMetaIntegrations();

  return {
    workflow: workflowQuery.data,
    workflowVersion: workflowVersionQuery.data,
    currentVersion:
      workflowQuery.data && workflowQuery.data.versions?.at(0)?.version,
    hasGithubIntegration: integrations?.some((i) => i.name === 'github'),
    deleteWorkflow,
    pauseWorkflow,
    unpauseWorkflow,
    workflowIsLoading: workflowQuery.isLoading,
    workflowVersionIsLoading: workflowVersionQuery.isLoading,
    isDeleting: deleteWorkflowMutation.isPending,
  };
};
