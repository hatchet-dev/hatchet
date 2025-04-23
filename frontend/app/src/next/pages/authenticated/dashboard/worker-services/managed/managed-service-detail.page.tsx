import { useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { WorkersProvider } from '@/next/hooks/use-workers';
import { Separator } from '@/next/components/ui/separator';
import { useBreadcrumbs } from '@/next/hooks/use-breadcrumbs';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  PageTitle,
  HeadlineActions,
  HeadlineActionItem,
} from '@/next/components/ui/page-header';
import docs from '@/next/docs-meta-data';
import { ROUTES } from '@/next/lib/routes';
import { WorkerDetailSheet } from '../components/worker-detail-sheet';
import { SheetViewLayout } from '@/next/components/layouts/sheet-view.layout';
import { WorkerDetailProvider } from '@/next/hooks/use-worker-detail';
import { WorkerType } from '@/lib/api';
import { ManagedComputeProvider } from '@/next/hooks/use-managed-compute';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/next/components/ui/tabs';
import { useManagedComputeDetail } from '@/next/hooks/use-managed-compute-detail';
import { ManagedComputeDetailProvider } from '@/next/hooks/use-managed-compute-detail';
import { UpdateServiceContent } from './components/update-service';
import { WorkersTab } from './components/workers-tab';
import { LogsTab } from './components/logs-tab';
import { MetricsTab } from './components/metrics-tab';
import { Badge } from '@/next/components/ui/badge';

function ServiceDetailPageContent({
  serviceId,
  workerId,
}: {
  serviceId: string;
  workerId: string;
}) {
  const navigate = useNavigate();

  const { data: service } = useManagedComputeDetail();

  useBreadcrumbs(
    () => [
      {
        title: 'Worker Services',
        label: service?.name || '',
        url: ROUTES.services.detail(
          encodeURIComponent(service?.metadata?.id || ''),
          WorkerType.MANAGED,
        ),
      },
    ],
    [service?.name, service?.metadata?.id],
  );

  const handleCloseSheet = useCallback(() => {
    navigate(
      ROUTES.services.detail(
        encodeURIComponent(service?.metadata?.id || ''),
        WorkerType.MANAGED,
      ),
    );
  }, [navigate, service?.metadata?.id]);

  return (
    <SheetViewLayout
      sheet={
        <WorkerDetailProvider workerId={workerId || ''}>
          <WorkerDetailSheet
            isOpen={!!workerId}
            onClose={handleCloseSheet}
            serviceName={service?.name || ''}
            workerId={workerId || ''}
          />
        </WorkerDetailProvider>
      }
    >
      <Headline>
        <PageTitle description="Manage workers in a worker service">
          {service?.name || ''} <Badge variant="outline">Managed</Badge>
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.workers} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <Tabs defaultValue="instances" className="w-full" state="query">
        <TabsList>
          <TabsTrigger value="instances">Workers</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
          <TabsTrigger value="builds">Builds & Deployments</TabsTrigger>
          <TabsTrigger value="metrics">Metrics</TabsTrigger>
          <TabsTrigger value="configuration">Configuration</TabsTrigger>
        </TabsList>
        <TabsContent value="instances">
          <WorkersTab serviceName={service?.name || ''} />
        </TabsContent>
        <TabsContent value="logs">
          <LogsTab />
        </TabsContent>
        {/* <TabsContent value="builds">
          <BuildsTab serviceName={service?.name || ''} />
        </TabsContent> */}
        <TabsContent value="metrics">
          <MetricsTab />
        </TabsContent>
        <TabsContent value="configuration">
          {service && <UpdateServiceContent />}
        </TabsContent>
      </Tabs>
    </SheetViewLayout>
  );
}

export default function ServiceDetailPage() {
  const { serviceName, workerId } = useParams<{
    serviceName: string;
    workerId?: string;
  }>();

  return (
    <ManagedComputeProvider>
      <WorkersProvider>
        <ManagedComputeDetailProvider
          managedWorkerId={serviceName || ''}
          defaultRefetchInterval={10000}
        >
          <ServiceDetailPageContent
            serviceId={serviceName || ''}
            workerId={workerId || ''}
          />
        </ManagedComputeDetailProvider>
      </WorkersProvider>
    </ManagedComputeProvider>
  );
}
