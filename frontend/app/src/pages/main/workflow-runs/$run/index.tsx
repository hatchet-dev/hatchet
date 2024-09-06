import { queries, WorkflowRunStatus } from '@/lib/api';
import { TenantContextType } from '@/lib/outlet';
import { useQuery } from '@tanstack/react-query';
import { useOutletContext, useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import RunDetailHeader from './components2/header';
import { WorkflowRunInputDialog } from './components2/workflow-run-input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { StepRunEvents } from './components2/step-run-events-for-workflow-run';
import { useState } from 'react';
import { MiniMap } from './components2/mini-map';
import { Sheet, SheetContent, SheetHeader } from '@/components/ui/sheet';
import StepRunDetail from './components2/step-run-detail/step-run-detail';

// import mock from './components2/mock.json';

export const WORKFLOW_RUN_TERMINAL_STATUSES = [
  WorkflowRunStatus.CANCELLED,
  WorkflowRunStatus.FAILED,
  WorkflowRunStatus.SUCCEEDED,
];

interface WorkflowRunSidebarState {
  stepRunId?: string;
}

export default function ExpandedWorkflowRun() {
  const [sidebarState, setSidebarState] = useState<WorkflowRunSidebarState>();

  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const params = useParams();
  invariant(params.run);

  const shape = useQuery({
    ...queries.workflowRuns.shape(tenant.metadata.id, params.run),
    refetchInterval: 1000,
  });

  // const shape = {
  //   isLoading: false,
  //   refetch: () => undefined,
  //   data: mock as unknown as WorkflowRunShape,
  // };

  return (
    <>
      <div className="p-8">
        <RunDetailHeader
          loading={shape.isLoading}
          data={shape.data}
          refetch={() => shape.refetch()}
        />
        {/* <div className="h-4" />
      <h2 className="text-lg font-semibold">Triggered By</h2>
      TODO -- trigger, timing
      <div className="h-4" />
      <h2 className="text-lg font-semibold">Errors</h2>
      <Alert variant="destructive" title="Error">
      TODO -- on failure & error aggregation
      </Alert>
      <div className="h-4" /> */}
        <div className="h-4" />
        {shape.data?.jobRuns?.map(({ job, stepRuns }, idx) => (
          <MiniMap
            steps={job?.steps}
            stepRuns={stepRuns}
            key={idx}
            selectedStepRunId={sidebarState?.stepRunId}
            onClick={(stepRunId) =>
              setSidebarState(
                stepRunId == sidebarState?.stepRunId
                  ? undefined
                  : { stepRunId },
              )
            }
          />
        ))}
        <div className="h-4" />
        <Tabs defaultValue="activity">
          <TabsList layout="underlined">
            <TabsTrigger value="activity">Activity</TabsTrigger>
            {/* <TabsTrigger value="logs">App Logs</TabsTrigger> */}
          </TabsList>
          <TabsContent value="activity">
            <div className="h-4" />
            {!shape.isLoading && shape.data && (
              <StepRunEvents
                workflowRun={shape.data}
                filteredStepRunId={sidebarState?.stepRunId}
                onClick={(stepRunId) =>
                  setSidebarState(
                    stepRunId == sidebarState?.stepRunId
                      ? undefined
                      : { stepRunId },
                  )
                }
              />
            )}
          </TabsContent>
          {/* <TabsContent value="logs">App Logs</TabsContent> */}
        </Tabs>
        <div className="h-4" />
        <h2 className="text-lg font-semibold">Workflow Run Input</h2>
        {shape.data && <WorkflowRunInputDialog run={shape.data} />}
      </div>
      {/* TODO drawer for mobile */}
      {shape.data && (
        <Sheet
          open={!!sidebarState}
          onOpenChange={(open) =>
            open ? undefined : setSidebarState(undefined)
          }
        >
          <SheetContent>
            <SheetHeader>
              {sidebarState?.stepRunId && (
                <StepRunDetail
                  stepRunId={sidebarState?.stepRunId}
                  workflowRun={shape.data}
                />
              )}
            </SheetHeader>
          </SheetContent>
        </Sheet>
      )}
    </>
  );
}
