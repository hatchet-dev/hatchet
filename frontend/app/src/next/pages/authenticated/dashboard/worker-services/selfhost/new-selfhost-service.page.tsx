import { WorkersProvider } from '@/next/hooks/use-workers';
import { useBreadcrumbs } from '@/next/hooks/use-breadcrumbs';
import { Headline, PageTitle } from '@/next/components/ui/page-header';
import docs from '@/next/lib/docs';
import { ROUTES } from '@/next/lib/routes';
import { WorkerType } from '@/lib/api';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { Separator } from '@/next/components/ui/separator';
import { baseDocsUrl } from '@/next/hooks/use-docs-sheet';
import { useEffect } from 'react';
import { useTenant } from '@/next/hooks/use-tenant';
function ServiceDetailPageContent() {
  const breadcrumb = useBreadcrumbs();
  const { tenant } = useTenant();

  useEffect(() => {
    breadcrumb.set([
      {
        title: 'Worker Services',
        label: 'New Selfhosted Service',
        url: ROUTES.services.new(
          tenant?.metadata?.id || '',
          WorkerType.SELFHOSTED,
        ),
      },
    ]);
  }, [breadcrumb]);

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Create workers that run in your own cloud or local environment.">
          New Selfhost Worker Service
        </PageTitle>
        {/* <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.workers} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions> */}
      </Headline>
      <Separator className="mt-4" />
      {/* FIXME: embed docs in the page natively */}
      <div className="flex flex-col gap-4">
        <iframe
          src={baseDocsUrl + docs.home.workers.href}
          className="w-full h-[calc(100vh-200px)] rounded-md border"
          title={`Documentation: ${docs.home.workers.title}`}
          loading="lazy"
        />
      </div>
    </BasicLayout>
  );
}

export default function ServiceDetailPage() {
  return (
    <WorkersProvider>
      <ServiceDetailPageContent />
    </WorkersProvider>
  );
}
