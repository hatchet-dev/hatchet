import { Separator } from '@/next/components/ui/separator';
import useApiMeta from '@/next/hooks/use-api-meta';
import useTenant from '@/next/hooks/use-tenant';
import { PaymentMethods } from './components/payment-methods';
import { DataTable } from '@/next/components/ui/data-table';
import { columns } from './components/resource-limit-columns';
import { Subscription } from './components/subscription';
import { HeadlineActionItem } from '@/next/components/ui/page-header';
import { HeadlineActions } from '@/next/components/ui/page-header';
import { PageTitle } from '@/next/components/ui/page-header';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { Headline } from '@/next/components/ui/page-header';
import { Button } from '@/next/components/ui/button';
import { ArrowUpRight } from 'lucide-react';
import { ROUTES } from '@/next/lib/routes';

export default function UsagePage() {
  const { cloud } = useApiMeta();
  const { limit } = useTenant();

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Manage your billing and resource limits">
          Usage
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <a href={ROUTES.common.pricing} target="_blank" rel="noreferrer">
              <Button>
                View Plan Details
                <ArrowUpRight className="h-4 w-4 mr-2" />
              </Button>
            </a>
            {/* <DocsButton doc={docs.home['github-integration']} size="icon" /> */}
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <div className="flex-grow h-full w-full">
        {cloud?.canBill && (
          <>
            <Separator className="my-4" />
            <PaymentMethods />
            <Separator className="my-4" />
            <Subscription />
            <Separator className="my-4" />
          </>
        )}

        <div className="flex flex-row justify-between items-center">
          <h3 className="text-xl font-semibold leading-tight text-foreground">
            Resource Limits
          </h3>
        </div>
        <p className="text-gray-700 dark:text-gray-300 my-4">
          Resource limits are used to control the usage of resources within a
          tenant. When a limit is reached, the system will take action based on
          the limit type. Please upgrade your plan, or{' '}
          <a
            href={ROUTES.common.contact}
            className="underline"
            target="_blank"
            rel="noreferrer"
          >
            contact us
          </a>{' '}
          if you need to adjust your limits.
        </p>

        <DataTable
          isLoading={limit?.isLoading}
          columns={columns()}
          data={limit?.data || []}
          filters={[]}
          getRowId={(row) => row.metadata.id}
        />
      </div>
    </BasicLayout>
  );
}
