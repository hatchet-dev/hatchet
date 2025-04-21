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
import { FilterProvider } from '@/next/hooks/use-filters';
import { PaginationProvider } from '@/next/hooks/use-pagination';
import { RunsProvider } from '@/next/hooks/use-runs';
import { Plus } from 'lucide-react';
import { useState } from 'react';
import { V1TaskStatus } from '@/lib/api';
import { SheetViewLayout } from '@/next/components/layouts/sheet-view.layout';
import { RunDetailSheet } from './run-detail-sheet';

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

  return (
    <SheetViewLayout
      sheet={
        taskId && (
          <RunDetailSheet
            isOpen={!!taskId}
            onClose={handleCloseSheet}
            workflowRunId={taskId.workflowRunId}
            taskId={taskId.taskId}
          />
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
      <FilterProvider
        initialFilters={{
          statuses: [
            V1TaskStatus.RUNNING,
            V1TaskStatus.COMPLETED,
            V1TaskStatus.FAILED,
          ],
          is_root_task: true,
        }}
      >
        <PaginationProvider>
          <RunsProvider>
            <GetWorkflowChart />
            <RunsTable
              rowClicked={(task) =>
                setTaskId({
                  workflowRunId:
                    task.workflowRunExternalId || task.taskExternalId,
                  taskId: task.taskExternalId,
                })
              }
              selectedTaskId={taskId?.taskId}
            />
          </RunsProvider>
        </PaginationProvider>
      </FilterProvider>
      <TriggerRunModal
        show={showTriggerModal}
        onClose={() => setShowTriggerModal(false)}
      />
    </SheetViewLayout>
  );
}
