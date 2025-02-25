import {
  V1TaskStatus,
  WorkflowRunShapeForWorkflowRunDetails,
  WorkflowRunStatus,
} from '@/lib/api';
import { useParams } from 'react-router-dom';
import { WorkflowRunInputDialog } from './v2components/workflow-run-input';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { StepRunEvents } from './v2components/step-run-events-for-workflow-run';
import { useCallback, useState } from 'react';
import StepRunDetail, {
  TabOption,
} from './v2components/step-run-detail/step-run-detail';
import { Separator } from '@/components/v1/ui/separator';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Sheet, SheetContent } from '@/components/v1/ui/sheet';
import { V1RunDetailHeader } from './v2components/header';
import { Badge } from '@/components/v1/ui/badge';
import { ViewToggle } from './v2components/view-toggle';
import WorkflowRunVisualizer from './v2components/workflow-run-visualizer-v2';
import { useAtom } from 'jotai';
import { preferredWorkflowRunViewAtom } from '@/lib/atoms';
import { JobMiniMap } from './v2components/mini-map';
import { useWorkflowDetails } from '../hooks';

export const WORKFLOW_RUN_TERMINAL_STATUSES = [
  WorkflowRunStatus.CANCELLED,
  WorkflowRunStatus.FAILED,
  WorkflowRunStatus.SUCCEEDED,
];

function statusToBadgeVariant(status: V1TaskStatus) {
  switch (status) {
    case V1TaskStatus.COMPLETED:
      return 'successful';
    case V1TaskStatus.FAILED:
    case V1TaskStatus.CANCELLED:
      return 'failed';
    default:
      return 'inProgress';
  }
}

const GraphView = ({
  shape,
  handleTaskRunExpand,
}: {
  shape: WorkflowRunShapeForWorkflowRunDetails;
  handleTaskRunExpand: (stepRunId: string) => void;
}) => {
  const [view] = useAtom(preferredWorkflowRunViewAtom);

  const showGraphView =
    view == 'graph' &&
    shape.some((task) => task.childrenExternalIds.length > 0);

  return showGraphView ? (
    <WorkflowRunVisualizer setSelectedTaskRunId={handleTaskRunExpand} />
  ) : (
    <JobMiniMap
      onClick={(stepRunId) => {
        if (stepRunId) {
          handleTaskRunExpand(stepRunId);
        }
      }}
    />
  );
};

export default function ExpandedWorkflowRun() {
  const params = useParams();
  const [selectedTaskRunId, setSelectedTaskRunId] = useState<string>();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  const handleTaskRunExpand = useCallback((taskRunId: string) => {
    setSelectedTaskRunId(taskRunId);
    setIsSidebarOpen(true);
  }, []);

  const { workflowRun, shape, isLoading, isError } = useWorkflowDetails();

  if (isLoading || isError || !workflowRun) {
    return null;
  }

  const inputData = JSON.stringify(workflowRun.input || {});
  const additionalMetadata = workflowRun.additionalMetadata;

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl pt-2 px-4 sm:px-6 lg:px-8">
        <V1RunDetailHeader />
        <Separator className="my-4" />
        <div className="flex flex-row gap-x-4 mb-4">
          <p className="font-semibold">Status</p>
          <Badge variant={statusToBadgeVariant(workflowRun.status)}>
            {workflowRun.status}
          </Badge>
        </div>
        <div className="w-full h-fit flex overflow-auto relative bg-slate-100 dark:bg-slate-900">
          <GraphView shape={shape} handleTaskRunExpand={handleTaskRunExpand} />
          <ViewToggle />
        </div>
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
            <StepRunEvents
              workflowRunId={params.run}
              fallbackTaskDisplayName={workflowRun.displayName}
              onClick={handleTaskRunExpand}
            />
          </TabsContent>
          <TabsContent value="input">
            <WorkflowRunInputDialog input={JSON.parse(inputData)} />
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
      <Sheet
        open={isSidebarOpen}
        onOpenChange={(open) => setIsSidebarOpen(open)}
      >
        <SheetContent className="w-fit min-w-[56rem] max-w-4xl sm:max-w-2xl z-[60]">
          {selectedTaskRunId && (
            <StepRunDetail
              taskRunId={selectedTaskRunId}
              defaultOpenTab={TabOption.Output}
              showViewTaskRunButton
            />
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
}
