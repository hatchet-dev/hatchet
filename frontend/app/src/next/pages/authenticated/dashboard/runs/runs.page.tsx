import BasicLayout from '@/next/components/layouts/basic.layout';
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
import { V1TaskStatus } from '@/next/lib/api';
import { Plus } from 'lucide-react';
import { useState } from 'react';

export default function RunsPage() {
  const [showTriggerModal, setShowTriggerModal] = useState(false);

  return (
    <BasicLayout>
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
            <RunsTable />
          </RunsProvider>
        </PaginationProvider>
      </FilterProvider>
      <TriggerRunModal
        show={showTriggerModal}
        onClose={() => setShowTriggerModal(false)}
      />
    </BasicLayout>
  );
}
