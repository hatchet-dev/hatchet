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
import { Button } from '@/next/components/ui/button';
import { Dialog, DialogContent } from '@/next/components/ui/dialog';
import { RotateCcw } from 'lucide-react';
import { Separator } from '@/next/components/ui/separator';
import { DangerZone } from './config/danger-zone';
import { ManagedServiceDetailTabs } from '../managed-service-detail.page';
import { WorkerType } from '@/lib/api';
interface SectionActionsProps {
  section: string;
  hasChanged: boolean;
  onRevert: () => void;
  onDeploy: () => void;
}

const SectionActions = ({
  section,
  hasChanged,
  onRevert,
  onDeploy,
}: SectionActionsProps) => {
  return (
    <div className="flex justify-end gap-2 p-4">
      {hasChanged && (
        <Button variant="outline" onClick={onRevert} className="gap-2">
          <RotateCcw className="h-4 w-4" />
          Revert
        </Button>
      )}
      <Button disabled={!hasChanged} onClick={onDeploy}>
        Deploy
      </Button>
    </div>
  );
};

export function UpdateServiceContent() {
  const navigate = useNavigate();
  const { data: service } = useManagedComputeDetail();
  const { update, delete: deleteService } = useManagedCompute();

  const [hasChanged, setHasChanged] = useState<Record<string, boolean>>({});
  const [showSummaryDialog, setShowSummaryDialog] = useState(false);

  // Initial values from service
  const initialGithubRepo: GithubRepoSelectorValue = {
    githubInstallationId: service?.buildConfig?.githubInstallationId || '',
    githubRepositoryOwner:
      service?.buildConfig?.githubRepository?.repo_owner || '',
    githubRepositoryName:
      service?.buildConfig?.githubRepository?.repo_name || '',
    githubRepositoryBranch: service?.buildConfig?.githubRepositoryBranch || '',
  };

  const initialBuildConfig: BuildConfigValue = {
    buildDir: service?.buildConfig?.steps?.[0]?.buildDir || './',
    dockerfilePath:
      service?.buildConfig?.steps?.[0]?.dockerfilePath || './Dockerfile',
    serviceName: service?.name || '',
  };

  const initialMachineConfig: MachineConfigValue = {
    cpuKind: service?.runtimeConfigs?.[0]?.cpuKind || 'shared',
    cpus: service?.runtimeConfigs?.[0]?.cpus || 1,
    memoryMb: service?.runtimeConfigs?.[0]?.memoryMb || 1024,
    regions: service?.runtimeConfigs?.[0]?.region
      ? [service.runtimeConfigs[0].region]
      : [ManagedWorkerRegion.Ewr],
    numReplicas: service?.runtimeConfigs?.[0]?.numReplicas,
    autoscaling: service?.runtimeConfigs?.[0]?.autoscaling,
  };

  const initialSecrets: UpdateManagedWorkerSecretRequest = {
    add: [],
    update: [],
    delete: [],
  };

  const [secrets, setSecrets] =
    useState<UpdateManagedWorkerSecretRequest>(initialSecrets);
  const [githubRepo, setGithubRepo] =
    useState<GithubRepoSelectorValue>(initialGithubRepo);
  const [buildConfig, setBuildConfig] =
    useState<BuildConfigValue>(initialBuildConfig);
  const [machineConfig, setMachineConfig] =
    useState<MachineConfigValue>(initialMachineConfig);

  const handleRevert = (section: string) => {
    switch (section) {
      case 'machineConfig':
        setMachineConfig(initialMachineConfig);
        break;
      case 'secrets':
        setSecrets(initialSecrets);
        break;
      case 'githubRepo':
        setGithubRepo(initialGithubRepo);
        break;
      case 'buildConfig':
        setBuildConfig(initialBuildConfig);
        break;
    }
    setHasChanged({
      ...hasChanged,
      [section]: false,
    });
  };

  const handleDeploy = async () => {
    if (!githubRepo.githubInstallationId || !githubRepo.githubRepositoryName) {
      return;
    }

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

      const to = ROUTES.services.detail(
        service?.metadata?.id || '',
        WorkerType.MANAGED,
        ManagedServiceDetailTabs.BUILDS,
      );

      console.log('to', to);
      navigate(to);
    } catch (error) {
      console.error('Failed to update service:', error);
    }
  };

  const handleDelete = async (serviceId: string) => {
    await deleteService.mutateAsync(serviceId);
    navigate(ROUTES.services.list);
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <dl className="flex flex-col gap-4">
        <MachineConfig
          config={machineConfig}
          setConfig={(value) => {
            setMachineConfig(value);
            setHasChanged({
              ...hasChanged,
              machineConfig: true,
            });
          }}
          actions={
            <SectionActions
              section="machineConfig"
              hasChanged={hasChanged.machineConfig}
              onRevert={() => handleRevert('machineConfig')}
              onDeploy={() => setShowSummaryDialog(true)}
            />
          }
          type="update"
        />
        <Separator />
        <EnvVarsEditor
          secrets={secrets}
          setSecrets={(value) => {
            setSecrets(value);
            setHasChanged({
              ...hasChanged,
              secrets: true,
            });
          }}
          original={{
            directSecrets: service?.directSecrets || [],
            globalSecrets: service?.globalSecrets || [],
          }}
          actions={
            <SectionActions
              section="secrets"
              hasChanged={hasChanged.secrets}
              onRevert={() => handleRevert('secrets')}
              onDeploy={() => setShowSummaryDialog(true)}
            />
          }
          type="update"
        />
        <Separator />
        <GithubIntegrationProvider>
          <GithubRepoSelector
            value={githubRepo}
            onChange={(value) => {
              setHasChanged({
                ...hasChanged,
                githubRepo: true,
              });
              setGithubRepo(value);
            }}
            actions={
              <SectionActions
                section="githubRepo"
                hasChanged={hasChanged.githubRepo}
                onRevert={() => handleRevert('githubRepo')}
                onDeploy={() => setShowSummaryDialog(true)}
              />
            }
            type="update"
          />
        </GithubIntegrationProvider>
        <Separator />
        <BuildConfig
          githubRepo={githubRepo}
          value={buildConfig}
          onChange={(value) => {
            setBuildConfig(value);
            setHasChanged({
              ...hasChanged,
              buildConfig: true,
            });
          }}
          type="update"
          actions={
            <SectionActions
              section="buildConfig"
              hasChanged={hasChanged.buildConfig}
              onRevert={() => handleRevert('buildConfig')}
              onDeploy={() => setShowSummaryDialog(true)}
            />
          }
        />
        <Separator />
        <DangerZone
          serviceName={service?.name || ''}
          serviceId={service?.metadata?.id || ''}
          onDelete={handleDelete}
          type="update"
        />
      </dl>

      <Dialog open={showSummaryDialog} onOpenChange={setShowSummaryDialog}>
        <DialogContent className="max-w-3xl">
          <Summary
            githubRepo={githubRepo}
            buildConfig={buildConfig}
            machineConfig={machineConfig}
            secrets={secrets}
            type="update"
            originalGithubRepo={initialGithubRepo}
            originalBuildConfig={initialBuildConfig}
            originalMachineConfig={initialMachineConfig}
          />
          <div className="pt-4 flex gap-2 justify-end">
            <Button
              variant="outline"
              onClick={() => setShowSummaryDialog(false)}
            >
              Cancel
            </Button>
            <Button loading={update.isPending} onClick={handleDeploy}>
              Deploy Changes
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
