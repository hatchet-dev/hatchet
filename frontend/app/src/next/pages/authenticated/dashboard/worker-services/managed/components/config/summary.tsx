import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/next/components/ui/card';
import { GithubRepoSelectorValue } from './github-repo-selector';
import { BuildConfigValue } from './build-config';
import { MachineConfigValue } from './machine-config/machine-config';
import { UpdateManagedWorkerSecretRequest } from '@/lib/api/generated/cloud/data-contracts';
import { regions } from './machine-config/types';
import { cn } from '@/lib/utils';

interface DiffValueProps {
  current: string | number;
  original?: string | number | undefined;
  className?: string;
  type: 'create' | 'update';
}

function DiffValue({ current, original, className, type }: DiffValueProps) {
  if (type === 'create') {
    return <span className={cn(className, 'text-green-600')}>{current}</span>;
  }

  if (!original || original === current) {
    return (
      <span className={cn(className, 'text-muted-foreground')}>{current}</span>
    );
  }

  return (
    <span className={cn(className, 'relative')}>
      <span className="line-through text-muted-foreground text-red-500">
        {original}
      </span>
      <span className="ml-2 text-green-600">{current}</span>
    </span>
  );
}

interface SummaryProps {
  githubRepo: GithubRepoSelectorValue;
  buildConfig: BuildConfigValue;
  machineConfig: MachineConfigValue;
  secrets: UpdateManagedWorkerSecretRequest;
  type: 'create' | 'update';
  originalGithubRepo?: GithubRepoSelectorValue;
  originalBuildConfig?: BuildConfigValue;
  originalMachineConfig?: MachineConfigValue;
  originalSecrets?: UpdateManagedWorkerSecretRequest;
}

export function Summary({
  githubRepo,
  buildConfig,
  machineConfig,
  secrets,
  type,
  originalGithubRepo,
  originalBuildConfig,
  originalMachineConfig,
}: SummaryProps) {
  return (
    <Card variant="borderless">
      <CardHeader>
        <CardTitle>Configuration Changeset</CardTitle>
        <CardDescription>
          Review your configuration changes before deploying
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <h3 className="font-medium">Repository</h3>
          <p className="text-sm text-muted-foreground">
            <DiffValue
              current={`${githubRepo.githubRepositoryOwner}/${githubRepo.githubRepositoryName} @ ${githubRepo.githubRepositoryBranch}`}
              original={
                originalGithubRepo
                  ? `${originalGithubRepo.githubRepositoryOwner}/${originalGithubRepo.githubRepositoryName} @ ${originalGithubRepo.githubRepositoryBranch}`
                  : undefined
              }
              type={type}
            />
          </p>
        </div>

        <div className="space-y-2">
          <h3 className="font-medium">Build Configuration</h3>
          <p className="text-sm text-muted-foreground">
            Service Name:{' '}
            <DiffValue
              current={buildConfig.serviceName}
              original={originalBuildConfig?.serviceName}
              type={type}
            />
          </p>
          <p className="text-sm text-muted-foreground">
            Build Directory:{' '}
            <DiffValue
              current={buildConfig.buildDir}
              original={originalBuildConfig?.buildDir}
              type={type}
            />
          </p>
          <p className="text-sm text-muted-foreground">
            Dockerfile:{' '}
            <DiffValue
              current={buildConfig.dockerfilePath}
              original={originalBuildConfig?.dockerfilePath}
              type={type}
            />
          </p>
        </div>

        <div className="space-y-2">
          <h3 className="font-medium">Machine Configuration</h3>
          <p className="text-sm text-muted-foreground">
            CPU:{' '}
            <DiffValue
              current={`${machineConfig.cpus} ${machineConfig.cpuKind}`}
              original={
                originalMachineConfig
                  ? `${originalMachineConfig.cpus} ${originalMachineConfig.cpuKind}`
                  : undefined
              }
              type={type}
            />
          </p>
          <p className="text-sm text-muted-foreground">
            Memory:{' '}
            <DiffValue
              current={`${machineConfig.memoryMb}MB`}
              original={
                originalMachineConfig
                  ? `${originalMachineConfig.memoryMb}MB`
                  : undefined
              }
              type={type}
            />
          </p>
          <p className="text-sm text-muted-foreground">
            Region:{' '}
            <DiffValue
              current={
                regions.find((r) => r.value === machineConfig.regions?.[0])
                  ?.name || 'Unknown'
              }
              original={
                originalMachineConfig
                  ? regions.find(
                      (r) => r.value === originalMachineConfig.regions?.[0],
                    )?.name || 'Unknown'
                  : undefined
              }
              type={type}
            />
          </p>
          {machineConfig.autoscaling ? (
            <>
              <p className="text-sm text-muted-foreground">
                Autoscaling:{' '}
                <DiffValue
                  current={`${machineConfig.autoscaling.minAwakeReplicas} - ${machineConfig.autoscaling.maxReplicas} replicas`}
                  original={
                    originalMachineConfig?.autoscaling
                      ? `${originalMachineConfig.autoscaling.minAwakeReplicas} - ${originalMachineConfig.autoscaling.maxReplicas} replicas`
                      : undefined
                  }
                  type={type}
                />
              </p>
              <p className="text-sm text-muted-foreground">
                Scale to Zero:{' '}
                <DiffValue
                  current={machineConfig.autoscaling.scaleToZero ? 'Yes' : 'No'}
                  original={
                    originalMachineConfig?.autoscaling
                      ? originalMachineConfig.autoscaling.scaleToZero
                        ? 'Yes'
                        : 'No'
                      : undefined
                  }
                  type={type}
                />
              </p>
              <p className="text-sm text-muted-foreground">
                Wait Duration:{' '}
                <DiffValue
                  current={machineConfig.autoscaling.waitDuration}
                  original={originalMachineConfig?.autoscaling?.waitDuration}
                  type={type}
                />
              </p>
              <p className="text-sm text-muted-foreground">
                Rolling Window:{' '}
                <DiffValue
                  current={machineConfig.autoscaling.rollingWindowDuration}
                  original={
                    originalMachineConfig?.autoscaling?.rollingWindowDuration
                  }
                  type={type}
                />
              </p>
              <p className="text-sm text-muted-foreground">
                Scale Up Threshold:{' '}
                <DiffValue
                  current={`${
                    machineConfig.autoscaling.utilizationScaleUpThreshold * 100
                  }%`}
                  original={
                    originalMachineConfig?.autoscaling
                      ? `${
                          originalMachineConfig.autoscaling
                            .utilizationScaleUpThreshold * 100
                        }%`
                      : undefined
                  }
                  type={type}
                />
              </p>
              <p className="text-sm text-muted-foreground">
                Scale Down Threshold:{' '}
                <DiffValue
                  current={`${
                    machineConfig.autoscaling.utilizationScaleDownThreshold *
                    100
                  }%`}
                  original={
                    originalMachineConfig?.autoscaling
                      ? `${
                          originalMachineConfig.autoscaling
                            .utilizationScaleDownThreshold * 100
                        }%`
                      : undefined
                  }
                  type={type}
                />
              </p>
              <p className="text-sm text-muted-foreground">
                Scaling Increment:{' '}
                <DiffValue
                  current={machineConfig.autoscaling.increment}
                  original={originalMachineConfig?.autoscaling?.increment}
                  type={type}
                />
              </p>
            </>
          ) : (
            <p className="text-sm text-muted-foreground">
              Static:{' '}
              <DiffValue
                current={`${machineConfig.numReplicas} replicas`}
                original={
                  originalMachineConfig
                    ? `${originalMachineConfig.numReplicas} replicas`
                    : undefined
                }
                type={type}
              />
            </p>
          )}
        </div>

        <div className="space-y-2">
          <h3 className="font-medium">Environment Variables</h3>
          {type === 'create' && (
            <p className={cn('text-sm', 'text-green-600')}>
              {secrets.add?.length || 0} new variables
            </p>
          )}
          {type === 'update' && (
            <>
              <p className={cn('text-sm', 'text-green-600')}>
                {secrets.add?.length || 0} new variables
              </p>
              <p
                className={cn(
                  'text-sm',
                  !secrets.update?.length || secrets.update?.length === 0
                    ? 'text-muted-foreground'
                    : 'text-yellow-500',
                )}
              >
                {secrets.update?.length || 0} updated variables
              </p>
              <p
                className={cn(
                  'text-sm',
                  !secrets.delete?.length || secrets.delete?.length === 0
                    ? 'text-muted-foreground'
                    : 'text-red-500',
                )}
              >
                {secrets.delete?.length || 0} deleted variables
              </p>
            </>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
