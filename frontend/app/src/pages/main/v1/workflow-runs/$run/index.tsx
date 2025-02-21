import { queries, V2TaskStatus, WorkflowRunStatus } from '@/lib/api';
import { TenantContextType } from '@/lib/outlet';
import { useQuery } from '@tanstack/react-query';
import { useOutletContext, useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import { WorkflowRunInputDialog } from './v2components/workflow-run-input';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { StepRunEvents } from './v2components/step-run-events-for-workflow-run';
import { useEffect, useState } from 'react';
import StepRunDetail, {
  TabOption,
} from './v2components/step-run-detail/step-run-detail';
import { Separator } from '@/components/v1/ui/separator';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Sheet, SheetContent } from '@/components/v1/ui/sheet';
import { V2RunDetailHeader } from './v2components/header';
import { Badge } from '@/components/v1/ui/badge';

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

function statusToBadgeVariant(status: V2TaskStatus) {
  switch (status) {
    case V2TaskStatus.COMPLETED:
      return 'successful';
    case V2TaskStatus.FAILED:
    case V2TaskStatus.CANCELLED:
      return 'failed';
    default:
      return 'inProgress';
  }
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

  const taskRunQuery = useQuery({
    ...queries.v2Tasks.get(params.run),
  });

  if (taskRunQuery.isLoading) {
    // TODO: Loading state
    return null;
  }

  const taskRun = taskRunQuery.data;

  if (!taskRun) {
    return null;
  }

  const inputData = taskRun.input;
  const additionalMetadata = taskRun.additionalMetadata;

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl pt-2 px-4 sm:px-6 lg:px-8">
        <V2RunDetailHeader taskRunId={params.run} />
        <Separator className="my-4" />
        <div className="flex flex-row gap-x-4">
          <p className="font-semibold">Status</p>
          <Badge variant={statusToBadgeVariant(taskRun.status)}>
            {taskRun.status}
          </Badge>
        </div>
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
                taskDisplayName={taskRun.displayName}
                onClick={(stepRunId) => {
                  setSidebarState(
                    stepRunId == sidebarState?.stepRunId
                      ? undefined
                      : { stepRunId, workflowRunId: params.run },
                  );
                }}
              />
            }
          </TabsContent>
          <TabsContent value="input">
            {inputData && (
              <WorkflowRunInputDialog input={JSON.parse(inputData)} />
            )}
          </TabsContent>
          <TabsContent value="additional-metadata">
            <CodeHighlighter
              className="my-4"
              language="json"
              code={JSON.stringify(additionalMetadata, null, 2)}
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
                taskRunId={params.run}
                defaultOpenTab={sidebarState?.defaultOpenTab}
              />
            )}
          </SheetContent>
        </Sheet>
      )}
    </div>
  );
}
