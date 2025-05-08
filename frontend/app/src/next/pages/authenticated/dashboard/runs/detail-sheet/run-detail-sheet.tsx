import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from '@/next/components/ui/tabs';
import { RunDetailProvider } from '@/next/hooks/use-run-detail';
import { TaskRunOverview } from './run-detail-summary';
import { RunDetailPayloadContent } from './run-detail-payloads';

export interface RunDetailSheetSerializableProps {
  pageWorkflowRunId?: string;
  selectedWorkflowRunId: string;
  selectedTaskId?: string;
  detailsLink?: string;
}

interface RunDetailSheetProps extends RunDetailSheetSerializableProps {
  isOpen: boolean;
  onClose: () => void;
}

export function RunDetailSheet(props: RunDetailSheetProps) {
  return <RunDetailProvider runId={props.selectedWorkflowRunId} defaultRefetchInterval={1000}>
    <RunDetailSheetContent {...props} />
  </RunDetailProvider>;
}

function RunDetailSheetContent({
  selectedTaskId,
  selectedWorkflowRunId,
  detailsLink,
}: RunDetailSheetProps) {
  return (
    <>
      <div className="flex flex-col gap-y-4 px-6 pt-12">
        {selectedWorkflowRunId}
        <TaskRunOverview selectedTaskId={selectedTaskId} detailsLink={detailsLink} />
        <Tabs
          defaultValue="payload"
          state="query"
          className="w-full"
        >
          <TabsList layout="underlined" className="w-full">
            <TabsTrigger variant="underlined" value="payload">
              Payloads
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="worker">
              Worker
            </TabsTrigger>
          </TabsList>
          <TabsContent value="payload" className="mt-4">
            <div className="flex flex-col gap-4">
              <RunDetailPayloadContent selectedTaskId={selectedTaskId} />
            </div>
          </TabsContent>
          <TabsContent value="worker" className="mt-4">
            {/* TODO: Add worker details */}
            {/* {selectedTask?.workerId ? (
            <WorkerDetails
              workerId={selectedTask}
              showActions={false}
            />
          ) : (
            <div className="text-center text-muted-foreground py-8">
              No worker information available
            </div>
          )} */}
          </TabsContent>
        </Tabs>
      </div>
    </>
  );
}
