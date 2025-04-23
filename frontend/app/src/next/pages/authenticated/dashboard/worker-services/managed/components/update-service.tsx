import { WorkerType } from '@/lib/api';
import {
  UpdateManagedWorkerSecretRequest,
  ManagedWorkerRegion,
} from '@/lib/api/generated/cloud/data-contracts';
import { GithubIntegrationProvider } from '@/next/hooks/use-github-integration';
import { useManagedCompute } from '@/next/hooks/use-managed-compute';
import { useManagedComputeDetail } from '@/next/hooks/use-managed-compute-detail';
import { ROUTES } from '@/next/lib/routes';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { BuildConfigValue, BuildConfig } from './config/build-config';
import { EnvVarsEditor } from './config/env-vars/env-vars';
import {
  GithubRepoSelectorValue,
  GithubRepoSelector,
} from './config/github-repo-selector';
import {
  MachineConfigValue,
  MachineConfig,
} from './config/machine-config/machine-config';
import { Summary } from './config/summary';

export function UpdateServiceContent() {
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
