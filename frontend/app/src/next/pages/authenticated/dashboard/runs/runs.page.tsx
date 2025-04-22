import GetWorkflowChart from '@/next/components/runs/runs-metrics/runs-histogram';
import { RunsTable } from '@/next/components/runs/runs-table/runs-table';
import { TriggerRunModal } from '@/next/components/runs/trigger-run-modal';
import { Button } from '@/next/components/ui/button';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  HeadlineActionItem,
  HeadlineActions,
  PageTitle,
} from '@/next/components/ui/page-header';
import { Separator } from '@/next/components/ui/separator';
import docs from '@/next/docs-meta-data';
import { RunsProvider } from '@/next/hooks/use-runs';
import { Plus } from 'lucide-react';
import { useState } from 'react';
import { V1TaskSummary } from '@/lib/api';
import { SheetViewLayout } from '@/next/components/layouts/sheet-view.layout';
import { RunDetailSheet } from './run-detail-sheet';
import { ROUTES } from '@/next/lib/routes';
import { TimeFilters } from '@/next/components/ui/filters/time-filter-group';
import { RunDetailProvider } from '@/next/hooks/use-run-detail';
import { RunsMetricsView } from '@/next/components/runs/runs-metrics/runs-metrics';
export interface RunsPageSheetProps {
  workflowRunId: string;
  taskId: string;
}

export default function RunsPage() {
  const [showTriggerModal, setShowTriggerModal] = useState(false);

  const [taskId, setTaskId] = useState<RunsPageSheetProps>();

  const handleCloseSheet = () => {
    setTaskId(undefined);
  };

  const handleRowClick = (task: V1TaskSummary) => {
    setTaskId({
      workflowRunId: task.workflowRunExternalId || task.taskExternalId,
      taskId: task.taskExternalId,
    });
  };

  return (
    <SheetViewLayout
      sheet={
        taskId && (
          <RunDetailProvider runId={taskId.workflowRunId}>
            <RunDetailSheet
              isOpen={!!taskId}
              onClose={handleCloseSheet}
              workflowRunId={taskId.workflowRunId}
              taskId={taskId.taskId}
              detailsLink={ROUTES.runs.taskDetail(
                taskId.workflowRunId,
                taskId.taskId,
              )}
            />
          </RunDetailProvider>
        )
      }
    >
      <Headline>
        <PageTitle description="View and filter runs on this tenant.">
          Runs
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home['running-tasks']} size="icon" />
          </HeadlineActionItem>
          <HeadlineActionItem>
            <Button onClick={() => setShowTriggerModal(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Trigger Run
            </Button>
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <RunsProvider>
        <dl className="flex flex-col gap-4 mt-4">
          <TimeFilters />
          <GetWorkflowChart />
          <RunsMetricsView />
          <RunsTable
            onRowClick={handleRowClick}
            selectedTaskId={taskId?.taskId}
          />
        </dl>
        <TriggerRunModal
          show={showTriggerModal}
          onClose={() => setShowTriggerModal(false)}
        />
      </RunsProvider>
    </SheetViewLayout>
  );
}
