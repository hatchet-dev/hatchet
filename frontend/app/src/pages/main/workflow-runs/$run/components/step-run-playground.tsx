import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import api, { StepRun, StepRunStatus, queries } from '@/lib/api';
import { useEffect, useState } from 'react';
import { RunStatus } from '../../components/run-statuses';
import { getTiming } from './step-run-node';
import { StepInputOutputSection } from './step-run-input-output';
import { Button } from '@/components/ui/button';
import invariant from 'tiny-invariant';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { cn } from '@/lib/utils';
import { useOutletContext } from 'react-router-dom';
import { TenantContextType } from '@/lib/outlet';

export function StepRunPlayground({
  stepRun,
  setStepRun,
}: {
  stepRun: StepRun | null;
  setStepRun: (stepRun: StepRun | null) => void;
}) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [errors, setErrors] = useState<string[]>([]);

  const { handleApiError } = useApiError({
    setErrors,
  });

  const originalInput = JSON.stringify(
    JSON.parse(stepRun?.input || '{}'),
    null,
    2,
  );

  const [stepInput, setStepInput] = useState(originalInput);

  useEffect(() => {
    setStepInput(originalInput);
  }, [originalInput]);

  const getStepRunQuery = useQuery({
    ...queries.stepRuns.get(tenant.metadata.id, stepRun?.metadata.id || ''),
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

  return (
    <Dialog
      open={!!stepRun}
      onOpenChange={(open) => {
        if (!open) {
          setStepRun(null);
        }
      }}
    >
      <DialogContent className="sm:max-w-[625px] py-12">
        <DialogHeader>
          <div className="flex flex-row justify-between items-center">
            <DialogTitle>
              {stepRun?.step?.readableId || stepRun?.metadata.id}
            </DialogTitle>
            <RunStatus status={stepRun?.status || StepRunStatus.PENDING} />
          </div>
          {stepRun && getTiming({ stepRun })}
          <DialogDescription>
            You can change the input to your step and see the output here. By
            default, this will trigger all child steps.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-row justify-between items-center">
          <div className="font-bold">Input</div>
          <Button
            className="w-fit"
            disabled={rerunStepMutation.isPending}
            onClick={() => {
              const inputObj = JSON.parse(stepInput);
              rerunStepMutation.mutate(inputObj);
            }}
          >
            <ArrowPathIcon
              className={cn(
                rerunStepMutation.isPending ? 'rotate-180' : '',
                'h-4 w-4 mr-2',
              )}
            />
            Rerun Step
          </Button>
        </div>
        {stepRun && (
          <StepInputOutputSection
            stepRun={stepRun}
            onInputChanged={(input: string) => {
              setStepInput(input);
            }}
          />
        )}
        {errors.length > 0 && (
          <div className="mt-4">
            {errors.map((error, index) => (
              <div key={index} className="text-red-500 text-sm">
                {error}
              </div>
            ))}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
