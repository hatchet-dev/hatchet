import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { WorkersProvider } from '@/next/hooks/use-workers';
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
import { WorkerType } from '@/lib/api';
import {
  useManagedCompute,
  ManagedComputeProvider,
} from '@/next/hooks/use-managed-compute';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { Separator } from '@/next/components/ui/separator';
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

function UpdateServicePageContent() {
  const { serviceName = '' } = useParams<{
    serviceName: string;
  }>();
  const navigate = useNavigate();

  const { data: services, update } = useManagedCompute();

  const decodedServiceName = decodeURIComponent(serviceName);

  const { setBreadcrumbs } = useBreadcrumbs();

  useEffect(() => {
    const breadcrumbs = [
      {
        title: 'Worker Services',
        label: serviceName,
        url: ROUTES.services.detail(decodedServiceName, WorkerType.MANAGED),
      },
      {
        title: 'Update Service',
        label: 'Update',
        url:
          ROUTES.services.detail(decodedServiceName, WorkerType.MANAGED) +
          '/update',
      },
    ];

    setBreadcrumbs(breadcrumbs);

    // Clear breadcrumbs when this component unmounts
    return () => {
      setBreadcrumbs([]);
    };
  }, [decodedServiceName, setBreadcrumbs, serviceName]);

  const service = services?.find((s) => s.name === decodedServiceName);

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

  if (!service) {
    return <div>Service not found</div>;
  }

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Update your managed worker service">
          Update Managed Worker Service
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.compute} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <dl className="flex flex-col gap-4">
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
          />
          <EnvVarsEditor
            secrets={secrets}
            setSecrets={setSecrets}
            original={{
              directSecrets: [],
              globalSecrets: [],
            }}
          />
          <MachineConfig
            config={machineConfig}
            setConfig={(value) => {
              setMachineConfig(value);
            }}
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
                service.buildConfig?.githubInstallationId || '',
              githubRepositoryOwner:
                service.buildConfig?.githubRepository?.repo_owner || '',
              githubRepositoryName:
                service.buildConfig?.githubRepository?.repo_name || '',
              githubRepositoryBranch:
                service.buildConfig?.githubRepositoryBranch || '',
            }}
            originalBuildConfig={{
              buildDir: service.buildConfig?.steps?.[0]?.buildDir || './',
              dockerfilePath:
                service.buildConfig?.steps?.[0]?.dockerfilePath ||
                './Dockerfile',
              serviceName: service.name || '',
            }}
            originalMachineConfig={{
              cpuKind: service.runtimeConfigs?.[0]?.cpuKind || 'shared',
              cpus: service.runtimeConfigs?.[0]?.cpus || 1,
              memoryMb: service.runtimeConfigs?.[0]?.memoryMb || 1024,
              regions: service.runtimeConfigs?.[0]?.region
                ? [service.runtimeConfigs[0].region]
                : [ManagedWorkerRegion.Ewr],
              numReplicas: service.runtimeConfigs?.[0]?.numReplicas,
              autoscaling: service.runtimeConfigs?.[0]?.autoscaling,
            }}
            originalSecrets={{
              add: [],
              update: [],
              delete: [],
            }}
          />
        </div>
      </div>
    </BasicLayout>
  );
}

export default function UpdateServicePage() {
  return (
    <ManagedComputeProvider>
      <WorkersProvider>
        <UpdateServicePageContent />
      </WorkersProvider>
    </ManagedComputeProvider>
  );
}
