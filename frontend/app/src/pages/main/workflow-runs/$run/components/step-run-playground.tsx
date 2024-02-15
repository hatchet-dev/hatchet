import api, { StepRun, StepRunStatus, queries } from '@/lib/api';
import { useEffect, useMemo, useState } from 'react';
import { RunStatus } from '../../components/run-statuses';
import { Button } from '@/components/ui/button';
import invariant from 'tiny-invariant';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { cn } from '@/lib/utils';
import { useOutletContext } from 'react-router-dom';
import { TenantContextType } from '@/lib/outlet';
import { PlayIcon } from '@radix-ui/react-icons';
import { StepRunOutput } from './step-run-output';
import { StepRunInputs } from './step-run-inputs';
import { Loading } from '@/components/ui/loading';
import { StepStatusDetails } from '..';

export function StepRunPlayground({
  stepRun,
  setStepRun,
  workflowRunId,
}: {
  stepRun: StepRun | undefined;
  setStepRun: (stepRun: StepRun | undefined) => void;
  workflowRunId: string;
}) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [errors, setErrors] = useState<string[]>([]);

  const { handleApiError } = useApiError({
    setErrors,
  });

  const originalInput = useMemo(
    () => JSON.stringify(JSON.parse(stepRun?.input || '{}'), null, 2),
    [stepRun?.input],
  );

  const [stepInput, setStepInput] = useState(originalInput);

  useEffect(() => {
    setStepInput(originalInput);
  }, [originalInput]);

  const getStepRunQuery = useQuery({
    ...queries.stepRuns.get(tenant.metadata.id, stepRun?.metadata.id || ''),
    enabled: !!stepRun,
    refetchInterval: (query) => {
      const data = query.state.data;

      if (
        data?.status != 'SUCCEEDED' &&
        data?.status != 'FAILED' &&
        data?.status != 'CANCELLED'
      ) {
        return 1000;
      }
    },
  });

  const queryClient = useQueryClient();

  const rerunStepMutation = useMutation({
    mutationKey: [
      'step-run:update:rerun',
      stepRun?.tenantId,
      stepRun?.metadata.id,
    ],
    mutationFn: async (input: object) => {
      invariant(stepRun?.tenantId, 'has tenantId');

      const res = await api.stepRunUpdateRerun(
        stepRun?.tenantId,
        stepRun?.metadata.id,
        {
          input: input,
        },
      );

      return res.data;
    },
    onMutate: () => {
      setErrors([]);
    },
    onSuccess: (stepRun: StepRun) => {
      queryClient.invalidateQueries({
        queryKey: queries.workflowRuns.get(tenant.metadata.id, workflowRunId)
          .queryKey,
      });

      setStepRun(stepRun);
      getStepRunQuery.refetch();
    },
    onError: handleApiError,
  });

  useEffect(() => {
    if (getStepRunQuery.data) {
      setStepRun(getStepRunQuery.data);
    }
  }, [getStepRunQuery.data, setStepRun]);

  const output = stepRun?.output || '{}';

  const COMPLETED = ['SUCCEEDED', 'FAILED', 'CANCELLED'];
  const isLoading = !COMPLETED.includes(stepRun?.status || '');

  const handleOnPlay = () => {
    const inputObj = JSON.parse(stepInput);
    rerunStepMutation.mutate(inputObj);
  };

  return (
    <div className="">
      {stepRun && (
        <>
          <div className="flex flex-row gap-4 mt-4">
            <div className="flex-grow w-1/2">
              {stepInput && (
                <StepRunInputs
                  schema={stepRun.inputSchema || ''}
                  input={stepInput}
                  setInput={setStepInput}
                  disabled={isLoading || rerunStepMutation.isPending}
                  handleOnPlay={handleOnPlay}
                />
              )}
            </div>
            <div className="flex-grow flex-col flex gap-4 w-1/2 ">
              <div className="flex flex-col sticky top-0">
                <div className="flex flex-row justify-between items-center mb-4">
                  <Button
                    className="w-fit"
                    disabled={isLoading || rerunStepMutation.isPending}
                    onClick={handleOnPlay}
                  >
                    {isLoading || rerunStepMutation.isPending ? (
                      <>
                        <Loading />
                        Playing
                      </>
                    ) : (
                      <>
                        <PlayIcon className={cn('h-4 w-4 mr-2')} />
                        Play Step
                      </>
                    )}
                  </Button>

                  <RunStatus
                    status={
                      errors.length > 0
                        ? StepRunStatus.FAILED
                        : stepRun?.status || StepRunStatus.PENDING
                    }
                  />
                </div>
                <StepRunOutput
                  stepRun={stepRun}
                  output={output}
                  isLoading={isLoading}
                  errors={
                    [
                      ...errors,
                      stepRun.error
                        ? StepStatusDetails({ stepRun })
                        : undefined,
                    ].filter((e) => !!e) as string[]
                  }
                />
              </div>
            </div>
          </div>
        </>
      )}
      {errors.length > 0 && <div className="mt-4"></div>}
    </div>
  );
}
