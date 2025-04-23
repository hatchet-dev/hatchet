import { useEffect, useCallback, useState } from 'react';
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
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/next/components/ui/tabs';
import { EnvVarsEditor } from './components/config/env-vars/env-vars';
import {
  ManagedWorkerRegion,
  UpdateManagedWorkerSecretRequest,
} from '@/lib/api/generated/cloud/data-contracts';
import {
  GithubRepoSelector,
  GithubRepoSelectorValue,
} from './components/config/github-repo-selector';
import { GithubIntegrationProvider } from '@/next/hooks/use-github-integration';
import {
  BuildConfig,
  BuildConfigValue,
} from './components/config/build-config';
import {
  MachineConfig,
  MachineConfigValue,
} from './components/config/machine-config/machine-config';
import { Summary } from './components/config/summary';
import { useManagedComputeDetail } from '@/next/hooks/use-managed-compute-detail';
import { ManagedComputeDetailProvider } from '@/next/hooks/use-managed-compute-detail';
function UpdateServiceContent() {
  const navigate = useNavigate();
  const { data: service } = useManagedComputeDetail();
  const { update } = useManagedCompute();

  const [secrets, setSecrets] = useState<UpdateManagedWorkerSecretRequest>({
    add: [],
    update: [],
    delete: [],
  });

  const [githubRepo, setGithubRepo] = useState<GithubRepoSelectorValue>({
    githubInstallationId: service?.buildConfig?.githubInstallationId || '',
    githubRepositoryOwner:
      service?.buildConfig?.githubRepository?.repo_owner || '',
    githubRepositoryName:
      service?.buildConfig?.githubRepository?.repo_name || '',
    githubRepositoryBranch: service?.buildConfig?.githubRepositoryBranch || '',
  });

  const [buildConfig, setBuildConfig] = useState<BuildConfigValue>({
    buildDir: service?.buildConfig?.steps?.[0]?.buildDir || './',
    dockerfilePath:
      service?.buildConfig?.steps?.[0]?.dockerfilePath || './Dockerfile',
    serviceName: service?.name || '',
  });

  const [machineConfig, setMachineConfig] = useState<MachineConfigValue>({
    cpuKind: service?.runtimeConfigs?.[0]?.cpuKind || 'shared',
    cpus: service?.runtimeConfigs?.[0]?.cpus || 1,
    memoryMb: service?.runtimeConfigs?.[0]?.memoryMb || 1024,
    regions: service?.runtimeConfigs?.[0]?.region
      ? [service.runtimeConfigs[0].region]
      : [ManagedWorkerRegion.Ewr],
    numReplicas: service?.runtimeConfigs?.[0]?.numReplicas,
    autoscaling: service?.runtimeConfigs?.[0]?.autoscaling,
  });

  const [isDeploying, setIsDeploying] = useState(false);

  const handleDeploy = async () => {
    if (!githubRepo.githubInstallationId || !githubRepo.githubRepositoryName) {
      return;
    }

    setIsDeploying(true);
    try {
      await update.mutateAsync({
        managedWorkerId: service?.metadata?.id || '',
        data: {
          name: buildConfig.serviceName,
          buildConfig: {
            ...githubRepo,
            steps: [
              {
                buildDir: buildConfig.buildDir,
                dockerfilePath: buildConfig.dockerfilePath,
              },
            ],
          },
          isIac: false,
          runtimeConfig: machineConfig,
          secrets: secrets,
        },
      });

      navigate(
        ROUTES.services.detail(buildConfig.serviceName, WorkerType.MANAGED),
      );
    } catch (error) {
      console.error('Failed to update service:', error);
    } finally {
      setIsDeploying(false);
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <dl className="flex flex-col gap-4">
        <MachineConfig
          config={machineConfig}
          setConfig={(value) => {
            setMachineConfig(value);
          }}
        />
        <EnvVarsEditor
          secrets={secrets}
          setSecrets={setSecrets}
          original={{
            directSecrets: service?.directSecrets || [],
            globalSecrets: service?.globalSecrets || [],
          }}
        />
        <GithubIntegrationProvider>
          <GithubRepoSelector
            value={githubRepo}
            onChange={(value) => {
              setGithubRepo(value);
            }}
          />
        </GithubIntegrationProvider>
        <BuildConfig
          githubRepo={githubRepo}
          value={buildConfig}
          onChange={(value) => {
            setBuildConfig(value);
          }}
          type="update"
        />
      </dl>
      <div className="sticky top-4 h-fit">
        <Summary
          githubRepo={githubRepo}
          buildConfig={buildConfig}
          machineConfig={machineConfig}
          secrets={secrets}
          onDeploy={handleDeploy}
          isDeploying={isDeploying}
          type="update"
          originalGithubRepo={{
            githubInstallationId:
              service?.buildConfig?.githubInstallationId || '',
            githubRepositoryOwner:
              service?.buildConfig?.githubRepository?.repo_owner || '',
            githubRepositoryName:
              service?.buildConfig?.githubRepository?.repo_name || '',
            githubRepositoryBranch:
              service?.buildConfig?.githubRepositoryBranch || '',
          }}
          originalBuildConfig={{
            buildDir: service?.buildConfig?.steps?.[0]?.buildDir || './',
            dockerfilePath:
              service?.buildConfig?.steps?.[0]?.dockerfilePath ||
              './Dockerfile',
            serviceName: service?.name || '',
          }}
          originalMachineConfig={{
            cpuKind: service?.runtimeConfigs?.[0]?.cpuKind || 'shared',
            cpus: service?.runtimeConfigs?.[0]?.cpus || 1,
            memoryMb: service?.runtimeConfigs?.[0]?.memoryMb || 1024,
            regions: service?.runtimeConfigs?.[0]?.region
              ? [service.runtimeConfigs[0].region]
              : [ManagedWorkerRegion.Ewr],
            numReplicas: service?.runtimeConfigs?.[0]?.numReplicas,
            autoscaling: service?.runtimeConfigs?.[0]?.autoscaling,
          }}
        />
      </div>
    </div>
  );
}

function ServiceDetailPageContent({
  serviceId,
  workerId,
}: {
  serviceId: string;
  workerId: string;
}) {
  const navigate = useNavigate();

  const { data: service } = useManagedComputeDetail();

  const { setBreadcrumbs } = useBreadcrumbs();

  useEffect(() => {
    const breadcrumbs = [
      {
        title: 'Worker Services',
        label: service?.name || '',
        url: ROUTES.services.detail(
          encodeURIComponent(service?.metadata?.id || ''),
          WorkerType.MANAGED,
        ),
      },
    ];

    setBreadcrumbs(breadcrumbs);

    // Clear breadcrumbs when this component unmounts
    return () => {
      setBreadcrumbs([]);
    };
  }, [setBreadcrumbs, service?.metadata?.id, service?.name]);

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
          {service?.name || ''}
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.workers} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <Tabs defaultValue="overview" className="w-full">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="update">Update Service</TabsTrigger>
        </TabsList>
        <TabsContent value="overview">
          {/* Stats Cards */}
          {/* <div className="mb-6">
            <WorkerStats stats={service} isLoading={isLoading} />
          </div> */}

          {/* Worker Table */}
          <WorkerTable serviceName={service?.name || ''} />
        </TabsContent>
        <TabsContent value="update">
          <UpdateServiceContent />
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
