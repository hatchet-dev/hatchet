import { RunDataCard } from '@/next/components/runs/run-output-card';
import { InfoSheet } from '@/next/components/ui/info-sheet';
import { RunId } from '@/next/components/runs/run-id';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/docs-meta-data';
import { V1WorkflowType } from '@/lib/api';
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from '@/components/v1/ui/tabs';
import { useSearchParams } from 'react-router-dom';
import { useCallback, useMemo } from 'react';
import { useRunDetail } from '@/next/hooks/use-run-detail';
import { Link } from 'react-router-dom';
import { ExternalLink } from 'lucide-react';

interface RunDetailSheetProps {
  isOpen: boolean;
  onClose: () => void;
  workflowRunId: string;
  taskId: string;
  detailsLink?: string;
}

export function RunDetailSheet({
  isOpen,
  onClose,
  taskId,
  detailsLink,
}: RunDetailSheetProps) {
  const { data } = useRunDetail();
  const workflow = useMemo(() => data?.run, [data]);

  const tasks = useMemo(() => data?.tasks, [data]);

  const selectedTask = useMemo(() => {
    if (taskId) {
      return tasks?.find((t) => t.taskExternalId === taskId);
    }
    return tasks?.[0];
  }, [tasks, taskId]);

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
    <InfoSheet
      isOpen={isOpen}
      onClose={onClose}
      title={
        selectedTask ? (
          <div className="flex flex-col gap-2">
            <div className="flex flex-row items-center justify-between gap-2">
              <span>Task Details</span>
              <RunId taskRun={selectedTask} />
            </div>
            {detailsLink && (
              <Link
                to={detailsLink}
                className="text-sm text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1"
              >
                View run details <ExternalLink className="h-3 w-3" />
              </Link>
            )}
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <span>Run:</span>
              <RunId wfRun={workflow} />
            </div>
            {detailsLink && (
              <Link
                to={detailsLink}
                className="text-sm text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1"
              >
                View run details <ExternalLink className="h-3 w-3" />
              </Link>
            )}
          </div>
        )
      }
    >
      <Tabs
        value={activeTab}
        onValueChange={handleTabChange}
        className="w-full"
      >
        <TabsList layout="underlined" className="w-full">
          <TabsTrigger variant="underlined" value="payload">
            Payload
          </TabsTrigger>
          <TabsTrigger variant="underlined" value="worker">
            Worker
          </TabsTrigger>
        </TabsList>
        <TabsContent value="payload" className="mt-4">
          <div className="flex flex-col gap-4">
            {selectedTask ? (
              <>
                <RunDataCard
                  title="Input"
                  output={(selectedTask.input as any).input}
                  variant="input"
                />
                {selectedTask.type === V1WorkflowType.DAG && (
                  <RunDataCard
                    title="Parent Data"
                    output={(selectedTask.input as any).parents}
                    variant="input"
                    collapsed
                  />
                )}
                <RunDataCard
                  title="Output"
                  output={selectedTask.output}
                  error={selectedTask.errorMessage}
                  status={selectedTask.status}
                  variant="output"
                />
                <RunDataCard
                  title="Metadata"
                  output={{
                    taskRunId: selectedTask.metadata.id,
                    workflowRunId: workflow?.metadata.id,
                    additional: selectedTask.additionalMetadata,
                  }}
                  status={selectedTask.status}
                  variant="metadata"
                  collapsed
                  actions={
                    <div className="flex items-center gap-2">
                      <DocsButton
                        doc={docs.home['additional-metadata']}
                        size="icon"
                      />
                    </div>
                  }
                />
              </>
            ) : (
              <>
                <RunDataCard
                  title="Input"
                  output={(workflow?.input as { input: object })?.input}
                  status={workflow?.status}
                  variant="input"
                />
                <RunDataCard
                  title="Metadata"
                  output={{
                    workflowRunId: workflow?.metadata.id,
                    additional: workflow?.additionalMetadata,
                  }}
                  status={workflow?.status}
                  variant="metadata"
                  collapsed
                  actions={
                    <div className="flex items-center gap-2">
                      <DocsButton
                        doc={docs.home['additional-metadata']}
                        size="icon"
                      />
                    </div>
                  }
                />
              </>
            )}
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
    </InfoSheet>
  );
}
