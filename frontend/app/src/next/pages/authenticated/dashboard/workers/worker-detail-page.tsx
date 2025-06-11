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
import BasicLayout from '@/next/components/layouts/basic.layout';
import { RunsProvider } from '@/next/hooks/use-runs';
import { RunsTable } from '@/next/components/runs/runs-table/runs-table';
import { V1TaskSummary } from '@/lib/api';
import { RunsMetricsView } from '@/next/components/runs/runs-metrics/runs-metrics';
import { TimeFilters } from '@/next/components/ui/filters/time-filter-group';
import { useWorker } from '@/next/hooks/use-worker';
import { Spinner } from '@/next/components/ui/spinner';
import { WorkerActions } from './components/actions';
import { useSidePanel } from '@/next/hooks/use-side-panel';
import { useState } from 'react';
import { SlotsBadge } from './components/worker-slots-badge';

function WorkerDetailPageContent() {
  const [selectedTaskId, setSelectedTaskId] = useState<string>();

  const { workerId } = useParams();

  const { data: workerDetails, isLoading: isWorkerLoading } = useWorker({
    workerId: workerId || '',
  });

  const { open: openSideSheet } = useSidePanel();

  const handleRowClick = (task: V1TaskSummary) => {
    setSelectedTaskId(task.taskExternalId);
    openSideSheet({
      type: 'run-details',
      content: {
        selectedWorkflowRunId: task.workflowRunExternalId,
        selectedTaskId: task.taskExternalId,
        pageWorkflowRunId: '',
      },
    });
  };

  if (isWorkerLoading) {
    return <Spinner />;
  }

  if (!workerDetails) {
    return <div>Worker not found</div>;
  }

  return (
    <BasicLayout>
      <Headline>
        <PageTitle
          description={`Viewing tasks executed by worker ${workerDetails.name} (${workerId})`}
        >
          {workerDetails.name}
          <SlotsBadge
            available={
              workerDetails.status === 'ACTIVE'
                ? workerDetails.availableRuns || 0
                : 0
            }
            max={workerDetails.maxRuns || 0}
          />{' '}
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
