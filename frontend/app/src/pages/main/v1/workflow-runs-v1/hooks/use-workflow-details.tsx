import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';

export const useWorkflowDetails = () => {
  const params = useParams();

  invariant(params.run);

  const { data, isLoading, isError } = useQuery({
    ...queries.v1WorkflowRuns.details(params.run),
  });

  const shape = data?.shape || [];
  const taskRuns = data?.tasks || [];
  const taskEvents = data?.taskEvents || [];
  const workflowRun = data?.run;

  return {
    shape,
    taskRuns,
    taskEvents,
    workflowRun,
    isLoading,
    isError,
  };
};
