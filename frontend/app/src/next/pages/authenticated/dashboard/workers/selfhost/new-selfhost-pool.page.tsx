import { WorkersProvider } from '@/next/hooks/use-workers';
import { Headline, PageTitle } from '@/next/components/ui/page-header';
import docs from '@/next/lib/docs';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { Separator } from '@/next/components/ui/separator';
import { baseDocsUrl } from '@/next/hooks/use-docs-sheet';

function WorkerPoolDetailPageContent() {
  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Create workers that run in your own cloud or local environment.">
          New Self-Hosted Worker
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

export default function WorkerPoolDetailPage() {
  return (
    <WorkersProvider>
      <WorkerPoolDetailPageContent />
    </WorkersProvider>
  );
}
