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
import BasicLayout from '@/next/components/layouts/basic.layout';
import { SlotsBadge, WorkerTable } from '../components';
import { WorkerActions } from '../components/actions';

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

  return (
    <BasicLayout>
      <Headline>
        <div className="flex flex-col gap-y-2">
          <PageTitle>
            {decodedPoolName}
            <SlotsBadge
              available={pool.totalAvailableRuns}
              max={pool.totalMaxRuns}
            />
          </PageTitle>
          <p className="text-muted-foreground">
            Viewing workers in pool{'  '}
            <code className="relative rounded bg-muted px-[0.3rem] py-[0.2rem] font-mono text-sm font-semibold">
              {decodedPoolName}
            </code>
          </p>
        </div>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.workers} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <WorkerTable poolName={pool.name} />
      <Separator className="my-4" />
      <WorkerActions actions={pool.actions} />
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
