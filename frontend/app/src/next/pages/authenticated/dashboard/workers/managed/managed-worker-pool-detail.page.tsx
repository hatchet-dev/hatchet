import { useParams, useSearchParams } from 'react-router-dom';
import { WorkersProvider } from '@/next/hooks/use-workers';
import { Separator } from '@/next/components/ui/separator';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  PageTitle,
  HeadlineActions,
  HeadlineActionItem,
} from '@/next/components/ui/page-header';
import docs from '@/next/lib/docs';
import { ManagedComputeProvider } from '@/next/hooks/use-managed-compute';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/next/components/ui/tabs';
import {
  useManagedComputeDetail,
  ManagedComputeDetailProvider,
} from '@/next/hooks/use-managed-compute-detail';
import { UpdateWorkerPoolContent } from './components/update-pool';
import { ManagedWorkerInstances } from './components/workers-tab';
import { LogsTab } from './components/logs-tab';
import { BuildsTab } from './components/builds-tab';
import { MetricsTab } from './components/metrics-tab';
import { Badge } from '@/next/components/ui/badge';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { WorkerTable } from '../components';

export enum ManagedWorkerPoolDetailTabs {
  INSTANCES = 'instances',
  LOGS = 'logs',
  BUILDS = 'builds',
  METRICS = 'metrics',
  CONFIGURATION = 'configuration',
}

function WorkerPoolDetailPageContent() {
  const { data: pool } = useManagedComputeDetail();
  const [searchParams, setSearchParams] = useSearchParams();
  const tabParam = searchParams.get('tab');
  const currentTab = tabParam || ManagedWorkerPoolDetailTabs.INSTANCES;

  const handleTabChange = (value: string) => {
    const newSearchParams = new URLSearchParams(searchParams);
    newSearchParams.set('tab', value);
    setSearchParams(newSearchParams);
  };

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="This managed worker pool is running on Hatchet.">
          {pool?.name || ''} <Badge variant="outline">Managed</Badge>
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.workers} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <Tabs
        value={currentTab}
        onValueChange={handleTabChange}
        className="w-full"
      >
        <TabsList>
          <TabsTrigger value={ManagedWorkerPoolDetailTabs.INSTANCES}>
            Instances
          </TabsTrigger>
          <TabsTrigger value={ManagedWorkerPoolDetailTabs.LOGS}>
            Logs
          </TabsTrigger>
          <TabsTrigger value={ManagedWorkerPoolDetailTabs.BUILDS}>
            Builds & Deployments
          </TabsTrigger>
          <TabsTrigger value={ManagedWorkerPoolDetailTabs.METRICS}>
            Metrics
          </TabsTrigger>
          <TabsTrigger value={ManagedWorkerPoolDetailTabs.CONFIGURATION}>
            Configuration
          </TabsTrigger>
        </TabsList>
        <TabsContent value="workers">
          <WorkerTable poolName={pool?.name || ''} />
        </TabsContent>
        <TabsContent value="instances">
          <ManagedWorkerInstances />
        </TabsContent>
        <TabsContent value="logs">
          <LogsTab />
        </TabsContent>
        <TabsContent value="builds">
          <BuildsTab />
        </TabsContent>
        <TabsContent value="metrics">
          <MetricsTab />
        </TabsContent>
        <TabsContent value="configuration">
          {pool ? <UpdateWorkerPoolContent /> : null}
        </TabsContent>
      </Tabs>
    </BasicLayout>
  );
}

export default function WorkerPoolDetailPage() {
  const { poolName } = useParams<{
    poolName: string;
  }>();

  return (
    <ManagedComputeProvider>
      <WorkersProvider>
        <ManagedComputeDetailProvider managedWorkerId={poolName || ''}>
          <WorkerPoolDetailPageContent />
        </ManagedComputeDetailProvider>
      </WorkersProvider>
    </ManagedComputeProvider>
  );
}
