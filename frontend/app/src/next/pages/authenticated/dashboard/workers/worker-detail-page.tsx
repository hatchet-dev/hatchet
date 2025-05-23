import { useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { WorkersProvider } from '@/next/hooks/use-workers';
import { Separator } from '@/next/components/ui/separator';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  PageTitle,
  HeadlineActions,
  HeadlineActionItem,
} from '@/next/components/ui/page-header';
import docs from '@/next/lib/docs';
import { Badge } from '@/next/components/ui/badge';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { RunsProvider } from '@/next/hooks/use-runs';
import { RunsTable } from '@/next/components/runs/runs-table/runs-table';
import { V1TaskSummary } from '@/lib/api';
import { useSideSheet } from '@/next/hooks/use-side-sheet';
import { RunsMetricsView } from '@/next/components/runs/runs-metrics/runs-metrics';
import { TimeFilters } from '@/next/components/ui/filters/time-filter-group';
import { useWorker } from '@/next/hooks/use-worker';
import { Spinner } from '@/next/components/ui/spinner';
import { WorkerActions } from './components/actions';

function WorkerDetailPageContent() {
  const { workerId } = useParams();

  const { data: workerDetails, isLoading: isWorkerLoading } = useWorker({
    workerId: workerId || '',
  });

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

  if (isWorkerLoading) {
    return <Spinner />;
  }

  if (!workerDetails) {
    return <div>Worker not found</div>;
  }

  const description = `Viewing tasks executed by worker ${workerId}`;

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description={description}>
          {workerDetails.name} <Badge variant="outline">Self-hosted</Badge>
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.workers} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <div className="flex flex-col gap-4 mt-4">
        <RunsProvider
          initialFilters={{
            worker_id: workerId,
            only_tasks: true,
          }}
          initialTimeRange={{
            activePreset: '24h',
          }}
        >
          <TimeFilters />
          <RunsMetricsView />
          <RunsTable
            onRowClick={handleRowClick}
            selectedTaskId={selectedTaskId}
          />
        </RunsProvider>
        <Separator className="my-4" />
        <WorkerActions actions={workerDetails.actions || []} />
      </div>
    </BasicLayout>
  );
}

export default function WorkerDetailPage() {
  return (
    <WorkersProvider>
      <WorkerDetailPageContent />
    </WorkersProvider>
  );
}
