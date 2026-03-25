import { queries, V1TaskStatus } from '@/lib/api';
import { getErrorStatus } from '@/lib/error-utils';
import { defaultQueryRetry } from '@/lib/query-retry';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';

export function isTerminalState(status: V1TaskStatus | undefined) {
  if (!status) {
    return false;
  }

  return [
    V1TaskStatus.COMPLETED,
    V1TaskStatus.FAILED,
    V1TaskStatus.CANCELLED,
  ].includes(status);
}

export const useWorkflowDetails = () => {
  const params = useParams({ from: appRoutes.tenantRunRoute.to });

  const { data, isLoading, isError, error } = useQuery({
    retry: defaultQueryRetry,
    refetchInterval: (query) => {
      const data = query.state.data;

      if (isTerminalState(data?.run?.status)) {
        return 5000;
      }

      return 1000;
    },
    ...queries.v1WorkflowRuns.details(params.run),
  });

  const shape = data?.shape || [];
  const taskRuns = data?.tasks || [];
  const taskEvents = data?.taskEvents || [];
  const workflowRun = data?.run;
  const workflowConfig = data?.workflowConfig;

  const errStatusCode = getErrorStatus(error);

  return {
    shape,
    taskRuns,
    taskEvents,
    workflowRun,
    workflowConfig,
    isLoading,
    isError,
    errStatusCode,
  };
};
