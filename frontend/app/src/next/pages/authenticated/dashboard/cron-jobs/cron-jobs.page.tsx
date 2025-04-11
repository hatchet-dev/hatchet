import { Separator } from '@/next/components/ui/separator';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  PageTitle,
  HeadlineActions,
  HeadlineActionItem,
} from '@/next/components/ui/page-header';
import docs from '@/next/docs-meta-data';
import CronJobsTable from './cron-jobs-table';
import useCan from '@/next/hooks/use-can';
import { cronJobs } from '@/next/lib/can/features/cron-jobs.permissions';
import { Alert, AlertTitle, AlertDescription } from '@/next/components/ui/alert';
import { Lock } from 'lucide-react';
import { PaginationProvider } from '@/next/components/ui/pagination';
import { FilterProvider } from '@/next/hooks/use-filters';

export default function CronJobsPage() {
  const { canWithReason } = useCan();
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
              <CronJobsTable />
            </PaginationProvider>
          </FilterProvider>
        </>
      )}
    </BasicLayout>
  );
}
