import { useEffect, useCallback, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useWorkers, WorkersProvider } from '@/next/hooks/use-workers';
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
import { ROUTES } from '@/next/lib/routes';
import { WorkerDetailSheet } from './components/worker-detail-sheet';
import { SheetViewLayout } from '@/next/components/layouts/sheet-view.layout';
import { WorkerDetailProvider } from '@/next/hooks/use-worker-detail';

function ServiceDetailPageContent() {
  const { serviceName = '', workerId } = useParams<{
    serviceName: string;
    workerId?: string;
  }>();
  const decodedServiceName = decodeURIComponent(serviceName);
  const navigate = useNavigate();

  const { services, isLoading } = useWorkers();

  const { setBreadcrumbs } = useBreadcrumbs();

  useEffect(() => {
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
  }, [services, decodedServiceName, setBreadcrumbs, serviceName]);

  const handleCloseSheet = useCallback(() => {
    navigate(ROUTES.services.detail(encodeURIComponent(decodedServiceName)));
  }, [navigate, decodedServiceName]);

  const service = useMemo(() => {
    console.log('Finding service:', decodedServiceName);
    console.log('Available services:', services);
    const found = services.find((s) => s.name === decodedServiceName);
    console.log('Found service:', found);
    return found;
  }, [services, decodedServiceName]);

  console.log('Services:', services);

  if (!service) {
    console.log('Service not found for:', decodedServiceName);
    return <div>Service not found</div>;
  }

  return (
    <SheetViewLayout
      sheet={
        <WorkerDetailProvider workerId={workerId || ''}>
          <WorkerDetailSheet
            isOpen={!!workerId}
            onClose={handleCloseSheet}
            serviceName={decodedServiceName}
            workerId={workerId || ''}
          />
        </WorkerDetailProvider>
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
      {/* Stats Cards */}
      <div className="mb-6">
        <WorkerStats stats={service} isLoading={isLoading} />
      </div>

      {/* Worker Table */}
      <WorkerTable serviceName={decodedServiceName} />
    </SheetViewLayout>
  );
}

export default function ServiceDetailPage() {
  return (
    <WorkersProvider>
      <ServiceDetailPageContent />
    </WorkersProvider>
  );
}
