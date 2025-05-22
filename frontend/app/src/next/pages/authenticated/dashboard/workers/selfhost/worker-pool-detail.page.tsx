import { useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { useWorkers, WorkersProvider } from '@/next/hooks/use-workers';
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

function WorkerPoolDetailPageContent() {
  const { poolName = '' } = useParams<{
    poolName: string;
  }>();
  const decodedPoolName = decodeURIComponent(poolName);
  const { pools } = useWorkers();

  const pool = useMemo(() => {
    return pools.find((s) => s.name === decodedPoolName);
  }, [pools, decodedPoolName]);

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

  if (!pool) {
    return <div>Worker not found</div>;
  }

  return (
    <BasicLayout>
      <Headline>
        <PageTitle
          description={`Viewing task runs executed by worker "${pool.name}"`}
        >
          {decodedPoolName} <Badge variant="outline">Self-hosted</Badge>
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.workers} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <div className="flex flex-col gap-4 mt-4">
        <TimeFilters />
        <RunsMetricsView />
        <RunsTable
          onRowClick={handleRowClick}
          selectedTaskId={selectedTaskId}
        />
      </div>
    </BasicLayout>
  );
}

export default function WorkerPoolDetailPage() {
  const { workerId } = useParams();

  return (
    <WorkersProvider>
      <RunsProvider
        initialFilters={{
          worker_id: workerId,
        }}
        initialTimeRange={{
          activePreset: '24h',
        }}
      >
        <WorkerPoolDetailPageContent />
      </RunsProvider>
    </WorkersProvider>
  );
}
