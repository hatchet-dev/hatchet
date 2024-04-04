import api, {
  PullRequestState,
  StepRun,
  StepRunStatus,
  WorkflowRun,
  queries,
} from '@/lib/api';
import { useEffect, useMemo, useState } from 'react';
import { Button } from '@/components/ui/button';
import invariant from 'tiny-invariant';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { cn } from '@/lib/utils';
import { useOutletContext } from 'react-router-dom';
import { TenantContextType } from '@/lib/outlet';
import { GitHubLogoIcon, PlayIcon } from '@radix-ui/react-icons';
import { StepRunOutput } from './step-run-output';
import { StepRunInputs } from './step-run-inputs';
import { Loading } from '@/components/ui/loading';
import { StepStatusDetails } from '..';
import {
  TooltipProvider,
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';
import { VscNote, VscJson } from 'react-icons/vsc';
import { CreatePRDialog } from './create-pr-dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { StepRunLogs } from './step-run-logs';
import { RunStatus } from '../../components/run-statuses';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { columns } from '../../components/workflow-runs-columns';
import { QuestionMarkCircleIcon, XMarkIcon } from '@heroicons/react/24/outline';

export function StepRunPlayground({
  stepRun,
  setStepRun,
  workflowRun,
}: {
  stepRun: StepRun | undefined;
  setStepRun: (stepRun: StepRun | undefined) => void;
  workflowRun: WorkflowRun;
}) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [errors, setErrors] = useState<string[]>([]);
  const [showPRDialog, setShowPRDialog] = useState(false);

  const { handleApiError } = useApiError({
    setErrors,
  });

  const updateParentData = (input: any, workflowRun: WorkflowRun) => {
    if (!input || !input.parents) {
      return {};
    }

    // HACK this is a temporary solution to get the parent data from the previous run
    // this should be handled by the backend
    const parents = Object.keys(input.parents);
    if (!workflowRun.jobRuns || !workflowRun.jobRuns[0]) {
      return input;
    }

    return workflowRun.jobRuns[0].stepRuns?.reduce((acc, stepRun) => {
      if (!stepRun.step || !parents.includes(stepRun.step.readableId)) {
        return acc;
      }

      return {
        ...acc,
        [stepRun.step.readableId]: JSON.parse(stepRun.output || '{}'),
      };
    }, {});
  };

  const originalInput = useMemo(() => {
    const input = JSON.parse(stepRun?.input || '{}');
    const previousRunData = updateParentData(input, workflowRun);

    const modifiedInput = JSON.stringify(
      {
        ...input,
        parents: {
          ...input.parents,
          ...previousRunData,
        },
      },
      null,
      2,
    );

    return modifiedInput;
  }, [stepRun?.input, workflowRun]);

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

      return 5000;
    },
  });

  const queryClient = useQueryClient();

  const stepRunSchemaQuery = useQuery({
    ...queries.stepRuns.getSchema(
      stepRun?.tenantId || '',
      stepRun?.metadata.id || '',
    ),
    enabled: !!stepRun,
    refetchInterval: () => {
      if (stepRun?.status === StepRunStatus.RUNNING) {
        return 1000;
      }

      return 5000;
    },
  });

  const stepRunDiffQuery = useQuery({
    ...queries.stepRuns.getDiff(stepRun?.metadata.id || ''),
    enabled: !!stepRun,
    refetchInterval: () => {
      if (stepRun?.status === StepRunStatus.RUNNING) {
        return 1000;
      }

      return 5000;
    },
  });

  const getWorkflowQuery = useQuery({
    ...queries.workflows.get(workflowRun?.workflowVersion?.workflowId || ''),
    enabled: !!workflowRun?.workflowVersion?.workflowId,
    refetchInterval: () => {
      if (stepRun?.status === StepRunStatus.RUNNING) {
        return 1000;
      }

      return 5000;
    },
  });

  const listPRsQuery = useQuery({
    ...queries.workflowRuns.listPullRequests(
      workflowRun.tenantId,
      workflowRun.metadata.id,
      {
        state: PullRequestState.Open,
      },
    ),
    enabled: !!stepRun,
    refetchInterval: () => {
      if (stepRun?.status === StepRunStatus.RUNNING) {
        return 1000;
      }

      return 5000;
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
      queryClient.invalidateQueries({
        queryKey: queries.workflowRuns.get(
          tenant.metadata.id,
          workflowRun.metadata.id,
        ).queryKey,
      });

      setStepRun(stepRun);
      getStepRunQuery.refetch();
    },
    onError: handleApiError,
  });

  const cancelStepMutation = useMutation({
    mutationKey: [
      'step-run:update:cancel',
      stepRun?.tenantId,
      stepRun?.metadata.id,
    ],
    mutationFn: async () => {
      invariant(stepRun?.tenantId, 'has tenantId');

      const res = await api.stepRunUpdateCancel(
        stepRun?.tenantId,
        stepRun?.metadata.id,
      );

      return res.data;
    },
    onMutate: () => {
      setErrors([]);
    },
    onSuccess: (stepRun: StepRun) => {
      queryClient.invalidateQueries({
        queryKey: queries.workflowRuns.get(
          tenant.metadata.id,
          workflowRun.metadata.id,
        ).queryKey,
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

  const handleOnCancel = () => {
    cancelStepMutation.mutate();
  };

  const [mode, setMode] = useState<'form' | 'json'>(
    (localStorage.getItem('mode') as 'form' | 'json') || 'form',
  );

  useEffect(() => {
    localStorage.setItem('mode', mode);
  }, [mode]);

  const handleModeSwitch = () => {
    setMode((prev) => (prev === 'json' ? 'form' : 'json'));
  };

  const disabled = rerunStepMutation.isPending || isLoading;

  // Function to detect the operating system
  const getOS = () => {
    const userAgent = window.navigator.userAgent;
    // Simple checks for platform; these could be extended as needed
    if (userAgent.includes('Mac')) {
      return 'MacOS';
    } else if (userAgent.includes('Win')) {
      return 'Windows';
    } else {
      // Default or other OS
      return 'unknown';
    }
  };
  // Determine the appropriate shortcut based on the OS
  const shortcut = getOS() === 'MacOS' ? 'Cmd + Enter' : 'Ctrl + Enter';

  const diffs = stepRunDiffQuery.data?.diffs || [];
  const workflow = getWorkflowQuery.data;
  const prs = listPRsQuery.data?.pullRequests || [];

  return (
    <div className="">
      {stepRun && (
        <>
          <div className="flex flex-row gap-2 justify-between items-center sticky top-0 z-50">
            <div className="text-2xl font-semibold tracking-tight">
              {stepRun?.step?.readableId}
            </div>
            <div className="flex flex-row gap-2 justify-end items-center">
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      size="icon"
                      onClick={handleModeSwitch}
                    >
                      {mode === 'json' && <VscNote className="h-4 w-4" />}
                      {mode === 'form' && <VscJson className="h-4 w-4" />}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {mode === 'json' && 'Switch to Form Mode'}
                    {mode === 'form' && 'Switch to JSON Mode'}
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      className="w-fit"
                      disabled={disabled}
                      onClick={handleOnPlay}
                    >
                      {disabled ? (
                        <>
                          <Loading />
                          Playing
                        </>
                      ) : (
                        <>
                          <PlayIcon className={cn('h-4 w-4 mr-2')} />
                          Replay Step
                        </>
                      )}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{shortcut}</TooltipContent>
                </Tooltip>
              </TooltipProvider>
              {diffs.length > 0 &&
                !!workflow?.deployment?.githubAppInstallationId &&
                prs.length == 0 && (
                  <TooltipProvider>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          className="w-fit"
                          disabled={disabled}
                          onClick={() => {
                            setShowPRDialog(true);
                          }}
                        >
                          <GitHubLogoIcon className={cn('h-4 w-4 mr-2')} />
                          Create Pull Request
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>
                        Create a new pull request to commit the changes that
                        you've made in the playground.
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                )}
              {prs.length > 0 && (
                <a
                  target="_blank"
                  href={`https://github.com/${prs[0].repositoryOwner}/${prs[0].repositoryName}/pull/${prs[0].pullRequestNumber}`}
                  rel="noreferrer"
                >
                  <Button className="w-fit" variant="ghost">
                    <GitHubLogoIcon className={cn('h-4 w-4 mr-2')} />
                    View Pull Request
                  </Button>
                </a>
              )}
            </div>
          </div>
          <div className="flex flex-row gap-4 mt-4">
            <div className="flex-grow w-1/2">
              <div className="text-lg font-semibold tracking-tight mb-4">
                Input
              </div>
              {stepInput && (
                <StepRunInputs
                  schema={stepRunSchemaQuery.data || {}}
                  input={stepInput}
                  setInput={setStepInput}
                  disabled={disabled}
                  handleOnPlay={handleOnPlay}
                  mode={mode}
                />
              )}
            </div>
            <div className="flex-grow flex-col flex gap-4 w-1/2 ">
              <Tabs defaultValue="output" className="flex flex-col">
                <div className="flex flex-row justify-between items-center">
                  <div className="flex flex-row justify-start items-center gap-6 mb-2">
                    <TabsList>
                      <TabsTrigger value="output" className="px-8">
                        Output
                      </TabsTrigger>
                      <TabsTrigger value="logs" className="px-8">
                        Logs
                      </TabsTrigger>
                    </TabsList>
                  </div>
                  <RunStatus
                    status={
                      errors.length > 0
                        ? StepRunStatus.FAILED
                        : stepRun?.status || StepRunStatus.PENDING
                    }
                  />
                </div>

                <TabsContent value="output">
                  <StepRunOutput
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
                </TabsContent>
                <TabsContent value="logs">
                  <StepRunLogs stepRun={stepRun} />
                </TabsContent>
              </Tabs>

              {isLoading && disabled && (
                <>
                  <Button className="w-fit" onClick={handleOnCancel}>
                    <>
                      <XMarkIcon className={cn('h-4 w-4 mr-2')} />
                      Attempt Cancel
                    </>
                  </Button>
                  <a href="https://docs.hatchet.run/home/features/cancellation">
                    <Button
                      onClick={handleOnCancel}
                      variant="link"
                      className="p-0 w-fit"
                      asChild
                    >
                      <>
                        <QuestionMarkCircleIcon
                          className={cn('h-4 w-4 mr-2')}
                        />
                        Help: How to handle cancelation signaling
                      </>
                    </Button>
                  </a>
                </>
              )}
            </div>
          </div>
        </>
      )}
      {stepRun &&
        stepRun.childWorkflowRuns &&
        stepRun.childWorkflowRuns.length > 0 && (
          <div className="flex flex-col gap-4 mt-4">
            <div className="text-lg font-semibold tracking-tight mb-4">
              Child Workflow Runs
            </div>
            <ChildWorkflowRuns stepRun={stepRun} workflowRun={workflowRun} />
          </div>
        )}
      {stepRun && workflowRun?.workflowVersion?.workflowId && (
        <CreatePRDialog
          show={showPRDialog}
          onClose={() => setShowPRDialog(false)}
          diffs={diffs}
          stepRun={stepRun}
          workflowId={workflowRun?.workflowVersion?.workflowId}
        />
      )}
      {errors.length > 0 && <div className="mt-4"></div>}
    </div>
  );
}

export function ChildWorkflowRuns({
  stepRun,
  workflowRun,
}: {
  stepRun: StepRun | undefined;
  workflowRun: WorkflowRun;
}) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const listWorkflowRunsQuery = useQuery({
    ...queries.workflowRuns.list(tenant.metadata.id, {
      parentWorkflowRunId: workflowRun.metadata.id,
      parentStepRunId: stepRun?.metadata.id,
    }),
    enabled: !!workflowRun && !!stepRun,
    refetchInterval: 5000,
  });

  if (listWorkflowRunsQuery.isLoading) {
    return <Loading />;
  }

  return (
    <DataTable
      columns={columns}
      data={listWorkflowRunsQuery.data?.rows || []}
      filters={[]}
    />
  );
}
