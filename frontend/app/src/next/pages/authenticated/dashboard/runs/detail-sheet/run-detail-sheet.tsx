import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from '@/components/v1/ui/tabs';
import { useSearchParams } from 'react-router-dom';
import { useCallback } from 'react';
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
  selectedTaskId: taskId,
  selectedWorkflowRunId: workflowRunId,
  detailsLink,
}: RunDetailSheetProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab = searchParams.get('task_tab') || 'payload';

  const handleTabChange = useCallback(
    (value: string) => {
      const newParams = new URLSearchParams(searchParams);
      newParams.set('task_tab', value);
      setSearchParams(newParams);
    },
    [searchParams, setSearchParams],
  );

  return (
    <>
      <div className="flex flex-col gap-y-4">
        <TaskRunOverview selectedTaskId={taskId} detailsLink={detailsLink} />
        <Tabs
          value={activeTab}
          onValueChange={handleTabChange}
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
              <RunDetailPayloadContent selectedTaskId={taskId} />
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
