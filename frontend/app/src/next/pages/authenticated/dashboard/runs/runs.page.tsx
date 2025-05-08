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
import docs from '@/next/lib/docs';
import { RunsProvider } from '@/next/hooks/use-runs';
import { Plus } from 'lucide-react';
import { useMemo, useState } from 'react';
import { V1TaskSummary } from '@/lib/api';
import { TimeFilters } from '@/next/components/ui/filters/time-filter-group';
import { RunsMetricsView } from '@/next/components/runs/runs-metrics/runs-metrics';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { useSideSheet } from '@/next/hooks/use-side-sheet';

export default function RunsPage() {
  const [showTriggerModal, setShowTriggerModal] = useState(false);


  const { open: openSideSheet, sheet} = useSideSheet();

  const handleRowClick = (task: V1TaskSummary) => {
    openSideSheet({
      type: 'task-detail',
      props: {
        selectedWorkflowRunId: task.workflowRunExternalId,
        selectedTaskId: task.taskExternalId,
        pageWorkflowRunId: ''
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
        <PageTitle description="View and filter runs on this tenant.">
          Runs
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.running_tasks} size="icon" />
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
        <div className="flex flex-col gap-4 mt-4">
          <TimeFilters />
          <GetWorkflowChart />
          <RunsMetricsView />
          <RunsTable
            onRowClick={handleRowClick}
            selectedTaskId={selectedTaskId}
            onTriggerRunClick={() => setShowTriggerModal(true)}
          />
        </div>
        <TriggerRunModal
          show={showTriggerModal}
          onClose={() => setShowTriggerModal(false)}
        />
      </RunsProvider>
    </BasicLayout>
  );
}
