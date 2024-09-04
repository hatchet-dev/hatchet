import { queries, StepRun, WorkflowRunStatus } from '@/lib/api';
import { TenantContextType } from '@/lib/outlet';
import { useQuery } from '@tanstack/react-query';
import { useOutletContext, useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import RunDetailHeader from './components2/header';
import { WorkflowRunInputDialog } from './components2/workflow-run-input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import WorkflowRunVisualizer from './components2/workflow-run-visualizer';
import { useState } from 'react';
import { MiniMap } from './components2/mini-map';
import { StepRunEvents } from './components/step-run-events';

export const WORKFLOW_RUN_TERMINAL_STATUSES = [
  WorkflowRunStatus.CANCELLED,
  WorkflowRunStatus.FAILED,
  WorkflowRunStatus.SUCCEEDED,
];

export default function ExpandedWorkflowRun() {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const params = useParams();
  invariant(params.run);

  const [selectedStepRun, setSelectedStepRun] = useState<StepRun | undefined>();

  const shape = useQuery({
    ...queries.workflowRuns.shape(tenant.metadata.id, params.run),
    refetchInterval: (query) => {
      const data = query.state.data;
      if (
        data &&
        data.jobRuns &&
        data.jobRuns.some((x) => x.status === 'RUNNING')
      ) {
        return 1000;
      }
    },
  });

  const state = useQuery({
    ...queries.workflowRuns.state(tenant.metadata.id, params.run),
    refetchInterval: 1000,
  });

  return (
    <div className="p-8">
      <RunDetailHeader loading={shape.isLoading} data={shape.data} />
      <div className="h-4" />
      <h2 className="text-lg font-semibold">Errors</h2>
      TODO -- on failure & error aggregation
      <div className="h-4" />
      <h2 className="text-lg font-semibold">Activity</h2>
      <StepRunEvents stepRun={undefined} />
      <div className="h-4" />
      <h2 className="text-lg font-semibold">Shape</h2>
      <Tabs defaultValue="overview">
        <TabsList layout="underlined">
          {shape.data?.jobRuns?.map((jobRun, idx) => (
            <TabsTrigger value={jobRun.jobId} key={idx}>
              {jobRun.job?.name}
            </TabsTrigger>
          ))}
        </TabsList>
        {shape.data?.jobRuns?.map((jobRun, idx) => (
          <TabsContent value={jobRun.jobId} key={idx}>
            <div className="w-full h-[200px] mt-8">
              value={jobRun.job?.name}
              {shape.data && (
                <WorkflowRunVisualizer
                  workflowRun={shape.data}
                  selectedStepRun={selectedStepRun}
                  setSelectedStepRun={(step) => {
                    setSelectedStepRun(
                      step.stepId === selectedStepRun?.stepId
                        ? undefined
                        : step,
                    );
                  }}
                />
              )}
            </div>
          </TabsContent>
        ))}
      </Tabs>
      {shape.data?.jobRuns?.map(({ job }, idx) => (
        <MiniMap steps={job?.steps} key={idx} />
      ))}
      <div className="h-4" />
      <h2 className="text-lg font-semibold">Input</h2>
      {shape.data && <WorkflowRunInputDialog run={shape.data} />}
      <pre>{JSON.stringify(shape.data, null, 2)}</pre>
      {/* <pre>{JSON.stringify(state.data, null, 2)}</pre> */}
    </div>
  );
}
