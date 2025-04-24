import { Table, TableBody, TableCell, TableRow } from '@/components/ui/table';

import { RunDataCard } from '@/next/components/runs/run-output-card';
import { InfoSheet } from '@/next/components/ui/info-sheet';
import { RunId } from '@/next/components/runs/run-id';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/docs-meta-data';
import { V1TaskStatus, V1WorkflowType } from '@/lib/api';
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
import { RunsBadge } from '@/next/components/runs/runs-badge';
import { Skeleton } from '@/next/components/ui/skeleton';

interface RunDetailSheetProps {
  isOpen: boolean;
  onClose: () => void;
  workflowRunId: string;
  taskId: string;
  detailsLink?: string;
}

type TaskRunSummaryTableProps = {
  status: V1TaskStatus;
  detailsLink?: string;
  runIdElement: JSX.Element;
};

const useTaskAndWorkflowSelection = ({
  taskId,
}: Pick<RunDetailSheetProps, 'taskId'>) => {
  const { data, isLoading } = useRunDetail();
  const workflow = useMemo(() => data?.run, [data]);

  const tasks = useMemo(() => data?.tasks, [data]);

  const selectedTask = useMemo(() => {
    if (taskId) {
      return tasks?.find((t) => t.taskExternalId === taskId);
    }
    return tasks?.[0];
  }, [tasks, taskId]);

  return {
    workflow,
    selectedTask,
    isLoading,
  };
};

const TaskRunSummaryTable = ({
  status,
  detailsLink,
  runIdElement,
}: TaskRunSummaryTableProps) => {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-2">
        <Table className="text-sm">
          <TableBody>
            <TableRow className="border-none">
              <TableCell className="pr-4 text-muted-foreground">
                Status
              </TableCell>
              <TableCell className="flex flex-row items-center justify-center">
                <RunsBadge status={status} variant="default" />
              </TableCell>
            </TableRow>
            <TableRow className="border-none">
              <TableCell className="pr-4 text-muted-foreground">
                Task ID
              </TableCell>
              <TableCell className="flex flex-row items-center justify-center">
                {runIdElement}
              </TableCell>
            </TableRow>
            {detailsLink && (
              <TableRow className="border-none hover:cursor-pointer">
                <TableCell className="pr-4 text-muted-foreground">
                  <Link
                    to={detailsLink}
                    className="text-sm text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1"
                  >
                    View Run Details
                    <ExternalLink className="h-3 w-3" />
                  </Link>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
};

const TaskRunOverview = ({
  taskId,
  detailsLink,
}: Pick<RunDetailSheetProps, 'detailsLink' | 'taskId'>) => {
  const { isLoading, workflow, selectedTask } = useTaskAndWorkflowSelection({
    taskId,
  });

  if (isLoading || !workflow) {
    return (
      <div className="size-full flex flex-col my-2 gap-y-4">
        <Skeleton className="h-28 w-full" />
        <div className="h-4" />
        <Skeleton className="h-36 w-full" />
        <Skeleton className="h-36 w-full" />
        <Skeleton className="h-36 w-full" />
      </div>
    );
  }

  if (!selectedTask) {
    return (
      <TaskRunSummaryTable
        status={workflow.status}
        detailsLink={detailsLink}
        runIdElement={<RunId wfRun={workflow} />}
      />
    );
  }

  return (
    <TaskRunSummaryTable
      status={selectedTask.status}
      detailsLink={detailsLink}
      runIdElement={<RunId taskRun={selectedTask} />}
    />
  );
};

const PayloadContent = ({ taskId }: Pick<RunDetailSheetProps, 'taskId'>) => {
  const { workflow, selectedTask } = useTaskAndWorkflowSelection({
    taskId,
  });

  if (!selectedTask) {
    return (
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
              <DocsButton doc={docs.home['additional-metadata']} size="icon" />
            </div>
          }
        />
      </>
    );
  }

  return (
    <>
      <RunDataCard
        title="Input"
        output={(selectedTask.input as any).input ?? {}}
        variant="input"
      />
      {selectedTask.type === V1WorkflowType.DAG && (
        <RunDataCard
          title="Parent Data"
          output={(selectedTask.input as any).parents ?? {}}
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
            <DocsButton doc={docs.home['additional-metadata']} size="icon" />
          </div>
        }
      />
    </>
  );
};

export function RunDetailSheet({
  isOpen,
  onClose,
  taskId,
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
    <InfoSheet
      isOpen={isOpen}
      onClose={onClose}
      title={
        <div className="flex flex-row items-center justify-between gap-2">
          <span>Task Run Details</span>
        </div>
      }
    >
      <div className="flex flex-col gap-y-4">
        <TaskRunOverview taskId={taskId} detailsLink={detailsLink} />
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
              <PayloadContent taskId={taskId} />
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
    </InfoSheet>
  );
}
