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
import { useMemo } from 'react';
import { V1TaskSummary } from '@/lib/api';
import { TimeFilters } from '@/next/components/ui/filters/time-filter-group';
import { RunsMetricsView } from '@/next/components/runs/runs-metrics/runs-metrics';
import { useParams } from 'react-router-dom';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { useSideSheet } from '@/next/hooks/use-side-sheet';

export default function EventsDetailPage() {
  const { eventId } = useParams<{
    eventId: string;
  }>();

  const { open: openSideSheet, sheet } = useSideSheet();

  const handleRowClick = (task: V1TaskSummary) => {
    openSideSheet({
      type: 'task-detail',
      props: {
        selectedWorkflowRunId: task.workflowRunExternalId,
        selectedTaskId: task.taskExternalId,
        pageWorkflowRunId: '',
      },
    });
  };

  const selectedTaskId = useMemo(() => {
    if (sheet?.openProps?.type === 'task-detail') {
      return sheet?.openProps?.props.selectedTaskId;
    }
    return undefined;
  }, [sheet]);

  return (
    <BasicLayout>
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
        initialFilters={{ triggering_event_external_id: eventId }}
        initialTimeRange={{
          activePreset: '7d',
        }}
      >
        <dl className="flex flex-col gap-4 mt-4">
          <TimeFilters />
          <RunsMetricsView />
          <RunsTable
            onRowClick={handleRowClick}
            selectedTaskId={selectedTaskId}
            excludedFilters={[
              'additional_metadata',
              'is_root_task',
              'statuses',
              'workflow_ids',
            ]}
          />
        </dl>
      </RunsProvider>
    </BasicLayout>
  );
}
