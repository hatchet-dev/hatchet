import { useCallback, useMemo } from 'react';
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
import docs from '@/next/lib/docs';
import { WorkerTable } from '../components/worker-table';
import { ROUTES } from '@/next/lib/routes';
import { WorkerDetailSheet } from '../components/worker-detail-sheet';
import { SheetViewLayout } from '@/next/components/layouts/sheet-view.layout';
import { WorkerDetailProvider } from '@/next/hooks/use-worker-detail';
import { Badge } from '@/next/components/ui/badge';
import { WorkerType } from '@/lib/api';

function ServiceDetailPageContent() {
  const { serviceName = '', workerName } = useParams<{
    serviceName: string;
    workerName?: string;
  }>();
  const decodedServiceName = decodeURIComponent(serviceName);
  const navigate = useNavigate();

  const { services, isLoading } = useWorkers();

  const service = useMemo(() => {
    return services.find((s) => s.name === decodedServiceName);
  }, [services, decodedServiceName]);

  useBreadcrumbs(
    () => [
      {
        title: 'Worker Services',
        label: serviceName,
        url: ROUTES.services.detail(
          encodeURIComponent(decodedServiceName),
          service?.type || WorkerType.SELFHOSTED,
        ),
      },
    ],
    [decodedServiceName, service?.type],
  );

  const handleCloseSheet = useCallback(() => {
    if (!service) {
      return;
    }

    navigate(
      ROUTES.services.detail(
        encodeURIComponent(decodedServiceName),
        service?.type,
      ),
    );
  }, [navigate, decodedServiceName, service]);

  if (!service) {
    return <div>Service not found</div>;
  }

  return (
    <SheetViewLayout
      sheet={
        <WorkerDetailProvider workerId={workerName || ''}>
          <WorkerDetailSheet
            isOpen={!!workerName}
            onClose={handleCloseSheet}
            serviceName={decodedServiceName}
            workerId={workerName || ''}
          />
        </WorkerDetailProvider>
      }
    >
      <Headline>
        <PageTitle description="Manage workers in a worker service">
          {decodedServiceName} <Badge variant="outline">Self-hosted</Badge>
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
