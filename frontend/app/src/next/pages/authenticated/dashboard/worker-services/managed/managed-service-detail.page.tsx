import { useEffect } from 'react';
import { useParams } from 'react-router-dom';
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
import docs from '@/next/lib/docs';
import { ROUTES } from '@/next/lib/routes';
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
import { Badge } from '@/next/components/ui/badge';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { useTenant } from '@/next/hooks/use-tenant';

export enum ManagedServiceDetailTabs {
  INSTANCES = 'instances',
  LOGS = 'logs',
  BUILDS = 'builds',
  METRICS = 'metrics',
  CONFIGURATION = 'configuration',
}

function ServiceDetailPageContent() {
  const { data: service } = useManagedComputeDetail();
  const { tenantId } = useTenant();

  const breadcrumb = useBreadcrumbs();

  useEffect(() => {
    breadcrumb.set([
      {
        title: 'Worker Services',
        label: service?.name || '',
        url: ROUTES.services.detail(
          tenantId,
          encodeURIComponent(service?.metadata?.id || ''),
          WorkerType.MANAGED,
        ),
      },
    ]);
  }, [service?.name, service?.metadata?.id, breadcrumb]);

  return (
    <BasicLayout>
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
      <Tabs
        defaultValue={ManagedServiceDetailTabs.INSTANCES}
        className="w-full"
        state="query"
      >
        <TabsList>
          <TabsTrigger value={ManagedServiceDetailTabs.INSTANCES}>
            Workers
          </TabsTrigger>
          <TabsTrigger value={ManagedServiceDetailTabs.LOGS}>Logs</TabsTrigger>
          <TabsTrigger value={ManagedServiceDetailTabs.BUILDS}>
            Builds & Deployments
          </TabsTrigger>
          <TabsTrigger value={ManagedServiceDetailTabs.METRICS}>
            Metrics
          </TabsTrigger>
          <TabsTrigger value={ManagedServiceDetailTabs.CONFIGURATION}>
            Configuration
          </TabsTrigger>
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
        <TabsContent value="metrics">{/* <MetricsTab /> */}</TabsContent>
        <TabsContent value="configuration">
          {service && <UpdateServiceContent />}
        </TabsContent>
      </Tabs>
    </BasicLayout>
  );
}

export default function ServiceDetailPage() {
  const { serviceName } = useParams<{
    serviceName: string;
    workerName?: string;
  }>();

  return (
    <ManagedComputeProvider>
      <WorkersProvider>
        <ManagedComputeDetailProvider managedWorkerId={serviceName || ''}>
          <ServiceDetailPageContent />
        </ManagedComputeDetailProvider>
      </WorkersProvider>
    </ManagedComputeProvider>
  );
}
