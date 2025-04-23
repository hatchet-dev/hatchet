import { createContext, useContext, useCallback, useMemo } from 'react';
import { useApiError, useApiMetaIntegrations } from '@/lib/hooks';
import api, { queries, WorkflowUpdateRequest } from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { FilterProvider } from './utils/use-filters';
import { PaginationProvider } from './utils/use-pagination';

interface WorkflowDetailsFilters {
  statuses?: string[];
  search?: string;
}

interface WorkflowDetailsState {
  workflow: any;
  workflowVersion: any;
  currentVersion: string | undefined;
  hasGithubIntegration: boolean;
  deleteWorkflow: () => void;
  pauseWorkflow: () => void;
  unpauseWorkflow: () => void;
  workflowIsLoading: boolean;
  workflowVersionIsLoading: boolean;
  isDeleting: boolean;
}

interface WorkflowDetailsProviderProps {
  children: React.ReactNode;
  workflowId: string;
}

const WorkflowDetailsContext = createContext<WorkflowDetailsState | null>(null);

export function useWorkflowDetails() {
  const context = useContext(WorkflowDetailsContext);
  if (!context) {
    throw new Error(
      'useWorkflowDetails must be used within a WorkflowDetailsProvider',
    );
  }
  return context;
}

function WorkflowDetailsProviderContent({
  children,
  workflowId,
}: WorkflowDetailsProviderProps) {
  const { handleApiError } = useApiError({});
  const navigate = useNavigate();
  const integrations = useApiMetaIntegrations();

  const workflowQuery = useQuery({
    ...queries.workflows.get(workflowId),
    refetchInterval: 1000,
  });

  const workflowVersionQuery = useQuery({
    ...queries.workflows.getVersion(workflowId),
    refetchInterval: 1000,
  });

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

  const pauseWorkflow = useCallback(() => {
    updateWorkflowMutation.mutate({ isPaused: true });
  }, [updateWorkflowMutation]);

  const unpauseWorkflow = useCallback(() => {
    updateWorkflowMutation.mutate({ isPaused: false });
  }, [updateWorkflowMutation]);

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

  const deleteWorkflow = useCallback(() => {
    deleteWorkflowMutation.mutate();
  }, [deleteWorkflowMutation]);

  const value = useMemo(
    () => ({
      workflow: workflowQuery.data,
      workflowVersion: workflowVersionQuery.data,
      currentVersion:
        workflowQuery.data && workflowQuery.data.versions?.at(0)?.version,
      hasGithubIntegration:
        integrations?.some((i) => i.name === 'github') || false,
      deleteWorkflow,
      pauseWorkflow,
      unpauseWorkflow,
      workflowIsLoading: workflowQuery.isLoading,
      workflowVersionIsLoading: workflowVersionQuery.isLoading,
      isDeleting: deleteWorkflowMutation.isPending,
    }),
    [
      workflowQuery.data,
      workflowQuery.isLoading,
      workflowVersionQuery.data,
      workflowVersionQuery.isLoading,
      integrations,
      deleteWorkflow,
      pauseWorkflow,
      unpauseWorkflow,
      deleteWorkflowMutation.isPending,
    ],
  );

  return (
    <WorkflowDetailsContext.Provider value={value}>
      {children}
    </WorkflowDetailsContext.Provider>
  );
}

export function WorkflowDetailsProvider({
  children,
  workflowId,
}: WorkflowDetailsProviderProps) {
  return (
    <FilterProvider<WorkflowDetailsFilters>
      initialFilters={{
        statuses: [],
        search: '',
      }}
    >
      <PaginationProvider initialPage={1} initialPageSize={50}>
        <WorkflowDetailsProviderContent workflowId={workflowId}>
          {children}
        </WorkflowDetailsProviderContent>
      </PaginationProvider>
    </FilterProvider>
  );
}
