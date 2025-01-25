import { queries, WorkflowRunStatus } from '@/lib/api';
import { TenantContextType } from '@/lib/outlet';
import { useQuery } from '@tanstack/react-query';
import { useOutletContext, useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import RunDetailHeader from './v2components/header';
import { WorkflowRunInputDialog } from './v2components/workflow-run-input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { StepRunEvents } from './v2components/step-run-events-for-workflow-run';
import { useEffect, useState } from 'react';
import { MiniMap } from './v2components/mini-map';
import { Sheet, SheetContent } from '@/components/ui/sheet';
import StepRunDetail, {
  TabOption,
} from './v2components/step-run-detail/step-run-detail';
import { Separator } from '@/components/ui/separator';
import { CodeHighlighter } from '@/components/ui/code-highlighter';
import WorkflowRunVisualizer from './v2components/workflow-run-visualizer-v2';
import { useAtom } from 'jotai';
import { preferredWorkflowRunViewAtom } from '@/lib/atoms';
import { hasChildSteps, ViewToggle } from './v2components/view-toggle';

export const WORKFLOW_RUN_TERMINAL_STATUSES = [
  WorkflowRunStatus.CANCELLED,
  WorkflowRunStatus.FAILED,
  WorkflowRunStatus.SUCCEEDED,
];

interface WorkflowRunSidebarState {
  workflowRunId?: string;
  stepRunId?: string;
  defaultOpenTab?: TabOption;
}

export default function ExpandedWorkflowRun() {
  const [sidebarState, setSidebarState] = useState<WorkflowRunSidebarState>();

  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const params = useParams();
  invariant(params.run);

  useEffect(() => {
    if (
      sidebarState?.workflowRunId &&
      params.run &&
      params.run !== sidebarState?.workflowRunId
    ) {
      setSidebarState(undefined);
    }
  }, [params.run, sidebarState]);

  const [view] = useAtom(preferredWorkflowRunViewAtom);

  console.log(params.run, tenant.metadata.id);

  const { data: _events, isLoading } = useQuery({
    ...queries.v2StepRunEvents.list(tenant.metadata.id, params.run, {
      limit: 50,
      offset: 0,
    }),
  });

  const events = _events?.rows || [];
  const inputData = events[0]?.data || {};

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl pt-2 px-4 sm:px-6 lg:px-8">
        {/* <RunDetailHeader
          loading={isLoading}
          data={shape.data}
          refetch={() => shape.refetch()}
        /> */}
        <Separator className="my-4" />
        {/* <div className="w-full h-fit flex overflow-auto relative bg-slate-100 dark:bg-slate-900">
          {shape.data && view == 'graph' && hasChildSteps(shape.data) && (
            <WorkflowRunVisualizer
              shape={shape.data}
              selectedStepRunId={sidebarState?.stepRunId}
              setSelectedStepRunId={(stepRunId) => {
                setSidebarState({
                  stepRunId,
                  defaultOpenTab: TabOption.Output,
                  workflowRunId: params.run,
                });
              }}
            />
          )}
          {shape.data && <ViewToggle shape={shape.data} />}
        </div> */}
        <div className="h-4" />
        <Tabs defaultValue="activity">
          <TabsList layout="underlined">
            <TabsTrigger variant="underlined" value="activity">
              Activity
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="input">
              Input
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="additional-metadata">
              Additional Metadata
            </TabsTrigger>
            {/* <TabsTrigger value="logs">App Logs</TabsTrigger> */}
          </TabsList>
          <TabsContent value="activity">
            <div className="h-4" />
            {
              <StepRunEvents
                taskRunId={params.run}
                onClick={(stepRunId) =>
                  setSidebarState(
                    stepRunId == sidebarState?.stepRunId
                      ? undefined
                      : { stepRunId, workflowRunId: params.run },
                  )
                }
              />
            }
          </TabsContent>
          <TabsContent value="input">
            {inputData && <WorkflowRunInputDialog run={inputData} />}
          </TabsContent>
          <TabsContent value="additional-metadata">
            <CodeHighlighter
              className="my-4"
              language="json"
              code={JSON.stringify(inputData, null, 2)}
            />
          </TabsContent>
        </Tabs>
      </div>
      {inputData && (
        <Sheet
          open={!!sidebarState}
          onOpenChange={(open) =>
            open ? undefined : setSidebarState(undefined)
          }
        >
          <SheetContent className="w-fit min-w-[56rem] max-w-4xl sm:max-w-2xl z-[60]">
            {sidebarState?.stepRunId && (
              <StepRunDetail
                stepRunId={sidebarState?.stepRunId}
                workflowRun={inputData}
                defaultOpenTab={sidebarState?.defaultOpenTab}
              />
            )}
          </SheetContent>
        </Sheet>
      )}
    </div>
  );
}
