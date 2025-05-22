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
import { WorkerTable } from '../components';

function WorkerPoolDetailPageContent() {
  const { poolName = '' } = useParams();

  const decodedPoolName = decodeURIComponent(poolName);
  const { pools } = useWorkers();

  const pool = useMemo(() => {
    return pools.find((s) => s.name === decodedPoolName);
  }, [pools, decodedPoolName]);

  if (!pool) {
    return <div>Worker not found</div>;
  }

  const description = `Viewing workers in pool "${pool.name}"`;

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description={description}>
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
        <WorkerTable poolName={pool.name} />
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
