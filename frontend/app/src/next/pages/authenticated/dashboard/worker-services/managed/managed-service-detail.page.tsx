import { useEffect, useCallback, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useWorkers, WorkersProvider } from '@/next/hooks/use-workers';
import { Separator } from '@/next/components/ui/separator';
import { useBreadcrumbs } from '@/next/hooks/use-breadcrumbs';
import { WorkerStats } from '../components';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  PageTitle,
  HeadlineActions,
  HeadlineActionItem,
} from '@/next/components/ui/page-header';
import docs from '@/next/docs-meta-data';
import { WorkerTable } from '../components/worker-table';
import { ROUTES } from '@/next/lib/routes';
import { WorkerDetailSheet } from '../components/worker-detail-sheet';
import { SheetViewLayout } from '@/next/components/layouts/sheet-view.layout';
import { WorkerDetailProvider } from '@/next/hooks/use-worker-detail';
import { WorkerType } from '@/lib/api';
import {
  ManagedComputeProvider,
  useManagedCompute,
} from '@/next/hooks/use-managed-compute';
import { useUnifiedWorkerServices } from '@/next/hooks/use-managed-compute';

function ServiceDetailPageContent() {
  const { serviceName = '', workerId } = useParams<{
    serviceName: string;
    workerId?: string;
  }>();
  const decodedServiceName = decodeURIComponent(serviceName);
  const navigate = useNavigate();

  const { isLoading: isRegularLoading } = useWorkers();
  const { isLoading: isManagedLoading } = useManagedCompute();

  const unifiedServices = useUnifiedWorkerServices();

  const { setBreadcrumbs } = useBreadcrumbs();

  useEffect(() => {
    const breadcrumbs = [
      {
        title: 'Worker Services',
        label: serviceName,
        url: ROUTES.services.detail(
          encodeURIComponent(decodedServiceName),
          WorkerType.MANAGED,
        ),
      },
    ];

    setBreadcrumbs(breadcrumbs);

    // Clear breadcrumbs when this component unmounts
    return () => {
      setBreadcrumbs([]);
    };
  }, [decodedServiceName, setBreadcrumbs, serviceName]);

  const handleCloseSheet = useCallback(() => {
    navigate(
      ROUTES.services.detail(
        encodeURIComponent(decodedServiceName),
        WorkerType.MANAGED,
      ),
    );
  }, [navigate, decodedServiceName]);

  const service = useMemo(() => {
    return unifiedServices.find((s) => s.name === decodedServiceName);
  }, [unifiedServices, decodedServiceName]);

  const isLoading = isRegularLoading || isManagedLoading;

  if (!service && !isLoading) {
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
      {service && (
        <div className="mb-6">
          <WorkerStats stats={service} isLoading={isLoading} />
        </div>
      )}

      {/* Worker Table */}
      <WorkerTable serviceName={decodedServiceName} />
    </SheetViewLayout>
  );
}

export default function ServiceDetailPage() {
  return (
    <ManagedComputeProvider>
      <WorkersProvider>
        <ServiceDetailPageContent />
      </WorkersProvider>
    </ManagedComputeProvider>
  );
}
