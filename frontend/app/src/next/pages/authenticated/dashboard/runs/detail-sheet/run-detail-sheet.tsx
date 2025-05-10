import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from '@/next/components/ui/tabs';
import { RunDetailProvider, useRunDetail } from '@/next/hooks/use-run-detail';
import { RunDetailPayloadContent } from './run-detail-payloads';
import { RunEventLog } from '@/next/components/runs/run-event-log/run-event-log';
import { useSideSheet } from '@/next/hooks/use-side-sheet';
import { useMemo } from 'react';
import { Button } from '@/next/components/ui/button';
import { AlertCircle, ArrowUpCircle } from 'lucide-react';
import { RunsBadge } from '@/next/components/runs/runs-badge';
import { RunId } from '@/next/components/runs/run-id';
import { Badge } from '@/next/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/next/components/ui/select';
import { cn } from '@/next/lib/utils';
import WorkflowRunVisualizer from '@/next/components/runs/run-dag/dag-run-visualizer';
import { TaskRunDetailProvider, useTaskRunDetail } from '@/next/hooks/use-task-run-detail';
import { RunDetailRawContent } from './run-detail-raw';
export interface RunDetailSheetSerializableProps {
  pageWorkflowRunId?: string;
  selectedWorkflowRunId: string;
  selectedTaskId?: string;
  detailsLink?: string;
  attempt?: number;
}

interface RunDetailSheetProps extends RunDetailSheetSerializableProps {
  isOpen: boolean;
  onClose: () => void;
}

export function RunDetailSheet(props: RunDetailSheetProps) {
  return <RunDetailProvider runId={props.selectedWorkflowRunId} defaultRefetchInterval={1000}>
    <TaskRunDetailProvider taskRunId={props.selectedTaskId || ''} attempt={props.attempt} defaultRefetchInterval={1000}>
      <RunDetailSheetContent {...props} />
    </TaskRunDetailProvider>
  </RunDetailProvider>;
}

function RunDetailSheetContent({
}: RunDetailSheetProps) {
  const { data } = useRunDetail();
  const { data: selectedTask } = useTaskRunDetail();
  const { open: openSheet, sheet } = useSideSheet();

  const { selectedTaskId, attempt } = sheet.openProps?.props as RunDetailSheetSerializableProps;

  const latestTask = useMemo(() => {
    return data?.tasks.find((task) => task.metadata.id === selectedTaskId);
  }, [data, selectedTaskId]);

  const populatedAttempt = useMemo(() => {
    return attempt || latestTask?.attempt || 0;
  }, [attempt, latestTask]);

  const isDAG = data?.shape.length && data?.shape.length > 1;

  return (
    <>
      <div className="h-full flex flex-col relative">
        <div className="sticky top-0 z-10 bg-yellow-500 bg-slate-100 dark:bg-slate-900 px-4 pb-2">
          <div className="flex flex-row items-center justify-between">
            <div className="flex flex-col gap-2 mt-4">
              <div className="flex flex-row items-center gap-2">
                <RunsBadge status={data?.run?.status} variant="xs" />
                <RunId wfRun={data?.run} onClick={()=>{
                  if (!data?.run?.metadata.id) return;
                  openSheet({
                    type: 'task-detail',
                    props: {
                      selectedWorkflowRunId: data.run.metadata.id,
                    },
                  });
                }}/>{isDAG && selectedTask && <>/</>}
              </div>
              {isDAG && selectedTask && (
                <div className="flex flex-row items-center gap-2">
                  <RunsBadge status={selectedTask?.status} variant="xs" />
                  <RunId wfRun={selectedTask} onClick={()=>{}}/>
                </div>
              )}
            </div>
            <div className="flex items-center gap-2">
              {isDAG ? (
                <>
                  <Badge variant="outline">DAG</Badge>
                </>
              ) : (
                <Badge variant="outline">Standalone</Badge>
              )}
            </div>
          </div>
          {latestTask?.attempt && populatedAttempt && (
            <div className="flex items-center justify-between text-sm">
              <div className={cn("flex items-center gap-1.5 text-yellow-700", populatedAttempt === latestTask.attempt && "text-green-700")}>
               { populatedAttempt !== latestTask.attempt && <><AlertCircle className="h-3.5 w-3.5" /> <span>Viewing attempt {populatedAttempt} of {latestTask.attempt}</span></>}
              </div>
              <div className="flex flex-row items-center gap-2">
                {latestTask.attempt > populatedAttempt && 
                <Button variant="link" size="sm" 
                tooltip="View latest attempt"
                disabled={populatedAttempt === latestTask.attempt} 
                onClick={() => {
                  if (!data?.run?.metadata.id || !selectedTask?.taskExternalId) return;
                  openSheet({
                    type: 'task-detail',
                    props: {
                      selectedWorkflowRunId: data.run.metadata.id,
                      selectedTaskId: selectedTask.taskExternalId,
                      attempt: latestTask.attempt,
                    },
                  });
                }}>
                  <ArrowUpCircle className="h-4 w-4" />
                </Button>
                }
              {selectedTask && (
                    <Select
                      value={populatedAttempt?.toString() || '0'}
                      onValueChange={(value) => {
                        if (!data?.run?.metadata.id || !selectedTask?.taskExternalId) return;
                        openSheet({
                          type: 'task-detail',
                          props: {
                            selectedWorkflowRunId: data.run.metadata.id,
                            selectedTaskId: selectedTask.taskExternalId,
                            attempt: parseInt(value),
                          },
                        });
                      }}
                    >
                      <SelectTrigger className="h-6 text-xs">
                        <SelectValue placeholder="Attempt" />
                      </SelectTrigger>
                      <SelectContent>
                        {Array.from({ length: (latestTask?.attempt || 0) }, (_, i) => i).reverse().map((i) => (
                          <SelectItem key={i} value={(i+1).toString()}>
                            Attempt {i + 1} {i + 1 == latestTask.attempt ? " (Current)" : ""}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
            </div>
            </div>
          )}
        </div>
        <div className="flex-1 overflow-y-scroll">
          <div className="bg-slate-100 dark:bg-slate-900">
            <WorkflowRunVisualizer
              workflowRunId={data?.run?.metadata.id || ''}
              patchTask={selectedTask}
            onTaskSelect={(taskId) => {
              if (!data?.run?.metadata.id) return;
              openSheet({
                type: 'task-detail',
                props: {
                  selectedWorkflowRunId: data.run.metadata.id,
                  selectedTaskId: taskId,
                  attempt: populatedAttempt,
                },
              });
            }}
            selectedTaskId={isDAG && selectedTask?.taskExternalId === data?.run?.metadata.id ? undefined : selectedTask?.taskExternalId}
          />
          </div>
          <Tabs
            defaultValue="payload"
            state="query"
            className="w-full"
          >
            <TabsList layout="underlined" className="w-full sticky top-0 z-10 bg-slate-100 dark:bg-slate-900">
              <TabsTrigger variant="underlined" value="payload">
                Payloads
              </TabsTrigger>
              <TabsTrigger variant="underlined" value="activity">
                Task Activity
              </TabsTrigger>
              <TabsTrigger variant="underlined" value="worker">
                Worker
              </TabsTrigger>
              <TabsTrigger variant="underlined" value="raw">
                Raw
              </TabsTrigger>
            </TabsList>
            <TabsContent value="activity" className="mt-4">
              {data?.run && <RunEventLog
                workflow={data.run}
                filters={{
                  taskId: [selectedTaskId || ''],
                  attempt: [populatedAttempt],
                }}
                showFilters={{
                  taskId: false,
                  attempt: false,
                }}
                onTaskSelect={(event) => {
                  console.log(event);
                  openSheet({
                    type: 'task-detail',
                    props: {
                      selectedWorkflowRunId: data.run.metadata.id,
                      selectedTaskId: event.taskId,
                      attempt: event.attempt,
                    },
                  });
                }}
              />}
            </TabsContent>
            <TabsContent value="payload" className="mt-4">
              <div className="flex flex-col gap-4">
                <RunDetailPayloadContent selectedTask={selectedTask} attempt={populatedAttempt} />
              </div>
            </TabsContent>
            <TabsContent value="worker" className="mt-4">
              {/* TODO: Add worker details */}
            </TabsContent>
            <TabsContent value="raw" className="mt-4">
              <RunDetailRawContent selectedTask={selectedTask} />
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </>
  );
}
