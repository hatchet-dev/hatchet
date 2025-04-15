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
import { WorkerDetails } from '@/next/pages/authenticated/dashboard/worker-services/components/worker-details';
import { useSearchParams } from 'react-router-dom';
import { useCallback } from 'react';

interface RunDetailSheetProps {
  isOpen: boolean;
  onClose: () => void;
  workflow: any;
  selectedTask?: any;
}

export function RunDetailSheet({
  isOpen,
  onClose,
  workflow,
  selectedTask,
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
    <InfoSheet
      isOpen={isOpen}
      onClose={onClose}
      title={
        selectedTask ? (
          <div className="flex items-center gap-2">
            <span>Task Details</span>
            <RunId taskRun={selectedTask} />
          </div>
        ) : (
          <div className="flex items-center gap-2">
            <span>Run:</span>
            <RunId wfRun={workflow} />
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
                    workflowRunId: workflow.metadata.id,
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
                  output={(workflow.input as { input: object }).input}
                  status={workflow.status}
                  variant="input"
                />
                <RunDataCard
                  title="Metadata"
                  output={{
                    workflowRunId: workflow.metadata.id,
                    additional: workflow.additionalMetadata,
                  }}
                  status={workflow.status}
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
          {selectedTask?.workerId ? (
            <WorkerDetails
              workerId={selectedTask.workerId}
              showActions={false}
            />
          ) : (
            <div className="text-center text-muted-foreground py-8">
              No worker information available
            </div>
          )}
        </TabsContent>
      </Tabs>
    </InfoSheet>
  );
}
