import { useMemo, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import useWorkers from '@/next/hooks/use-workers';
import { Separator } from '@/next/components/ui/separator';
import { useBreadcrumbs } from '@/next/hooks/use-breadcrumbs';
import { WorkerStats } from './components';
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
import { WorkerDetailSheet } from './components/worker-detail-sheet';
import { SheetViewLayout } from '@/next/components/layouts/sheet-view.layout';

export default function ServiceDetailPage() {
  const { serviceName = '', workerId } = useParams<{
    serviceName: string;
    workerId?: string;
  }>();
  const decodedServiceName = decodeURIComponent(serviceName);
  const navigate = useNavigate();

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
        title: 'Worker Services',
        label: serviceName,
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

  const handleCloseSheet = useCallback(() => {
    navigate(ROUTES.services.detail(encodeURIComponent(decodedServiceName)));
  }, [navigate, decodedServiceName]);

  return (
    <SheetViewLayout
      sheet={
        <WorkerDetailSheet
          isOpen={!!workerId}
          onClose={handleCloseSheet}
          serviceName={decodedServiceName}
          workerId={workerId || ''}
        />
      }
    >
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
    </SheetViewLayout>
  );
}
