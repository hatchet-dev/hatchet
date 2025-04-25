import { Separator } from '@/next/components/ui/separator';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  PageTitle,
  HeadlineActions,
  HeadlineActionItem,
} from '@/next/components/ui/page-header';
import docs from '@/next/lib/docs';
import CronJobsTable from './cron-jobs-table';
import useCan from '@/next/hooks/use-can';
import { cronJobs } from '@/next/lib/can/features/cron-jobs.permissions';
import {
  Alert,
  AlertTitle,
  AlertDescription,
} from '@/next/components/ui/alert';
import { Lock, Plus } from 'lucide-react';
import { Button } from '@/next/components/ui/button';
import { useState } from 'react';
import { TriggerRunModal } from '@/next/components/runs/trigger-run-modal';
import { CronsProvider } from '@/next/hooks/use-crons';

export default function CronJobsPage() {
  const { canWithReason } = useCan();
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);

  const { allowed: canManage, message: canManageMessage } = canWithReason(
    cronJobs.manage(),
  );

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Schedule recurring task runs based on cron expressions">
          Cron Jobs
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home['cron-runs']} size="icon" />
          </HeadlineActionItem>
          {canManage && (
            <HeadlineActionItem>
              <Button
                className="w-full md:w-auto"
                onClick={() => setIsCreateDialogOpen(true)}
              >
                <Plus className="h-4 w-4 mr-2" />
                Create Cron Job
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
          <CronsProvider>
            <CronJobsTable
              onCreateClicked={() => setIsCreateDialogOpen(true)}
            />
          </CronsProvider>

          <TriggerRunModal
            show={isCreateDialogOpen}
            onClose={() => setIsCreateDialogOpen(false)}
            defaultTimingOption="cron"
          />
        </>
      )}
    </BasicLayout>
  );
}
