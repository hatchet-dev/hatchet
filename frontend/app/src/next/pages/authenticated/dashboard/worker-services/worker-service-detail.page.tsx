import { useMemo, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import useWorkers from '@/next/hooks/use-workers';
import { Separator } from '@/next/components/ui/separator';
import { useBreadcrumbs } from '@/next/hooks/use-breadcrumbs';
import { WorkerStats } from './components';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  PageTitle,
  HeadlineActions,
  HeadlineActionItem,
} from '@/next/components/ui/page-header';
import docs from '@/next/docs-meta-data';
import { WorkerTable } from './components/worker-table';
import { FilterProvider } from '@/next/hooks/use-filters';
import { ROUTES } from '@/next/lib/routes';

export default function ServiceDetailPage() {
  const { serviceName = '' } = useParams<{ serviceName: string }>();
  const decodedServiceName = decodeURIComponent(serviceName);

  const { data: workers = [], isLoading } = useWorkers({
    refetchInterval: 5000,
  });

  const { setBreadcrumbs } = useBreadcrumbs();

  useEffect(() => {
    if (!workers) {
      return;
    }

    const breadcrumbs = [
      {
        title: serviceName,
        url: ROUTES.services.detail(encodeURIComponent(decodedServiceName)),
      },
    ];

    setBreadcrumbs(breadcrumbs);

    // Clear breadcrumbs when this component unmounts
    return () => {
      setBreadcrumbs([]);
    };
  }, [workers, decodedServiceName, setBreadcrumbs, serviceName]);

  // Filter workers for this service
  const serviceWorkers = useMemo(() => {
    return workers.filter((worker) => worker.name === decodedServiceName);
  }, [workers, decodedServiceName]);

  // Service stats
  const serviceStats = useMemo(() => {
    return {
      total: serviceWorkers.length,
      active: serviceWorkers.filter((worker) => worker.status === 'ACTIVE')
        .length,
      inactive: serviceWorkers.filter((worker) => worker.status === 'INACTIVE')
        .length,
      paused: serviceWorkers.filter((worker) => worker.status === 'PAUSED')
        .length,
      slots: serviceWorkers
        .filter((worker: any) => worker.status === 'ACTIVE')
        .reduce((sum: number, worker: any) => sum + (worker.maxRuns || 0), 0),
      maxSlots: serviceWorkers
        .filter((worker: any) => worker.status === 'ACTIVE')
        .reduce((sum: number, worker: any) => sum + (worker.maxRuns || 0), 0),
      lastActive: null,
    };
  }, [serviceWorkers]);

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Manage workers in a worker service">
          {decodedServiceName}
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.workers} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <FilterProvider>
        {/* Stats Cards */}
        <div className="mb-6">
          <WorkerStats stats={serviceStats} isLoading={isLoading} />
        </div>

        {/* Worker Table */}
        <WorkerTable serviceName={decodedServiceName} />
      </FilterProvider>
    </BasicLayout>
  );
}
