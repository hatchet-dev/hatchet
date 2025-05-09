import { RunsTable } from '@/next/components/runs/runs-table/runs-table';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  HeadlineActionItem,
  HeadlineActions,
  PageTitle,
} from '@/next/components/ui/page-header';
import { Separator } from '@/next/components/ui/separator';
import docs from '@/next/lib/docs';
import { RunsProvider } from '@/next/hooks/use-runs';
import { useState } from 'react';
import { V1TaskSummary } from '@/lib/api';
import { SheetViewLayout } from '@/next/components/layouts/sheet-view.layout';
import { RunDetailSheet } from '../runs/run-detail-sheet';
import { ROUTES } from '@/next/lib/routes';
import { TimeFilters } from '@/next/components/ui/filters/time-filter-group';
import { RunDetailProvider } from '@/next/hooks/use-run-detail';
import { RunsMetricsView } from '@/next/components/runs/runs-metrics/runs-metrics';
import { useParams } from 'react-router-dom';

interface RunsPageSheetProps {
  workflowRunId: string;
  taskId: string;
}

export default function EventsDetailPage() {
  const { eventId } = useParams<{
    eventId: string;
  }>();

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
        <PageTitle description={`Viewing runs triggered by event ${eventId}`}>
          Runs
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.run_on_event} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <RunsProvider
        initialFilters={{ triggering_event_id: eventId }}
        initialTimeRange={{
          activePreset: '7d',
        }}
      >
        <dl className="flex flex-col gap-4 mt-4">
          <TimeFilters />
          <RunsMetricsView />
          <RunsTable
            onRowClick={handleRowClick}
            selectedTaskId={taskId?.taskId}
            excludedFilters={[
              'additional_metadata',
              'is_root_task',
              'statuses',
              'workflow_ids',
            ]}
          />
        </dl>
      </RunsProvider>
    </SheetViewLayout>
  );
}
