import { ScheduledRunsTable } from './components/scheduled-runs-table';
import useCan from '@/next/hooks/use-can';
import { scheduledRuns } from '@/next/lib/can/features/scheduled-runs.permissions';
import {
  Alert,
  AlertTitle,
  AlertDescription,
} from '@/next/components/ui/alert';
import { Lock, Plus } from 'lucide-react';
import { PaginationProvider } from '@/next/components/ui/pagination';
import { FilterProvider } from '@/next/hooks/use-filters';
import {
  Headline,
  HeadlineActionItem,
  HeadlineActions,
  PageTitle,
} from '@/next/components/ui/page-header';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/docs-meta-data';
import { Separator } from '@/next/components/ui/separator';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { Button } from '@/next/components/ui/button';
import { useState } from 'react';
import { TriggerRunModal } from '@/next/components/runs/trigger-run-modal';

export default function ScheduledRunsPage() {
  const { canWithReason } = useCan();
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);

  const { allowed: canManage, message: canManageMessage } = canWithReason(
    scheduledRuns.manage(),
  );

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Run tasks at a specific date and time">
          Scheduled Runs
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home['scheduled-runs']} size="icon" />
          </HeadlineActionItem>
          {canManage && (
            <HeadlineActionItem>
              <Button
                className="w-full md:w-auto"
                onClick={() => setIsCreateDialogOpen(true)}
              >
                <Plus className="h-4 w-4 mr-2" />
                Schedule New Run
              </Button>
            </HeadlineActionItem>
          )}
        </HeadlineActions>
      </Headline>
      {canManageMessage && (
        <Alert variant="warning">
          <Lock className="w-4 h-4 mr-2" />
          <AlertTitle>Role required</AlertTitle>
          <AlertDescription>{canManageMessage}</AlertDescription>
        </Alert>
      )}
      {canManage && (
        <>
          <Separator className="my-4" />

          <FilterProvider>
            <PaginationProvider>
              <ScheduledRunsTable
                onCreateClicked={() => setIsCreateDialogOpen(true)}
              />
            </PaginationProvider>
          </FilterProvider>

          <TriggerRunModal
            show={isCreateDialogOpen}
            onClose={() => setIsCreateDialogOpen(false)}
            defaultTimingOption="schedule"
          />
        </>
      )}
    </BasicLayout>
  );
}
