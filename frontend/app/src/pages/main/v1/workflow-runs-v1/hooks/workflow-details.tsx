import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { AxiosError, isAxiosError } from 'axios';
import { useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';

export const useWorkflowDetails = () => {
  const params = useParams();

  invariant(params.run);

  const { data, isLoading, isError, error } = useQuery({
    retry: (_f, error: AxiosError) => {
      if (error.response?.status === 404) {
        return false;
      }

      return true;
    },
    ...queries.v1WorkflowRuns.details(params.run),
  });

  const shape = data?.shape || [];
  const taskRuns = data?.tasks || [];
  const taskEvents = data?.taskEvents || [];
  const workflowRun = data?.run;
  let errStatusCode: number | undefined;

  // get the status code of the error
  if (error && isAxiosError(error)) {
    const axiosErr = error as AxiosError;

    errStatusCode = axiosErr.response?.status;
  }

  return {
    shape,
    taskRuns,
    taskEvents,
    workflowRun,
    isLoading,
    isError,
    errStatusCode,
  };
};
