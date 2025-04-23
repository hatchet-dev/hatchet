import { Button } from '@/next/components/ui/button';
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

interface SummaryProps {
  githubRepo: GithubRepoSelectorValue;
  buildConfig: BuildConfigValue;
  machineConfig: MachineConfigValue;
  secrets: UpdateManagedWorkerSecretRequest;
  onDeploy: () => void;
  isDeploying?: boolean;
}

export function Summary({
  githubRepo,
  buildConfig,
  machineConfig,
  secrets,
  onDeploy,
  isDeploying = false,
}: SummaryProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Configuration Summary</CardTitle>
        <CardDescription>
          Review your configuration before deploying
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <h3 className="font-medium">Repository</h3>
          <p className="text-sm text-muted-foreground">
            {githubRepo.githubRepositoryOwner}/{githubRepo.githubRepositoryName}{' '}
            @ {githubRepo.githubRepositoryBranch}
          </p>
        </div>

        <div className="space-y-2">
          <h3 className="font-medium">Build Configuration</h3>
          <p className="text-sm text-muted-foreground">
            Service Name: {buildConfig.serviceName}
          </p>
          <p className="text-sm text-muted-foreground">
            Build Directory: {buildConfig.buildDir}
          </p>
          <p className="text-sm text-muted-foreground">
            Dockerfile: {buildConfig.dockerfilePath}
          </p>
        </div>

        <div className="space-y-2">
          <h3 className="font-medium">Machine Configuration</h3>
          <p className="text-sm text-muted-foreground">
            CPU: {machineConfig.cpus} {machineConfig.cpuKind}
          </p>
          <p className="text-sm text-muted-foreground">
            Memory: {machineConfig.memoryMb}MB
          </p>
          <p className="text-sm text-muted-foreground">
            Region:{' '}
            {regions.find((r) => r.value === machineConfig.regions?.[0])?.name}
          </p>
          {machineConfig.autoscaling ? (
            <>
              <p className="text-sm text-muted-foreground">
                Autoscaling: {machineConfig.autoscaling.minAwakeReplicas} -{' '}
                {machineConfig.autoscaling.maxReplicas} replicas
              </p>
              <p className="text-sm text-muted-foreground">
                Scale to Zero:{' '}
                {machineConfig.autoscaling.scaleToZero ? 'Yes' : 'No'}
              </p>
              <p className="text-sm text-muted-foreground">
                Wait Duration: {machineConfig.autoscaling.waitDuration}
              </p>
              <p className="text-sm text-muted-foreground">
                Rolling Window:{' '}
                {machineConfig.autoscaling.rollingWindowDuration}
              </p>
              <p className="text-sm text-muted-foreground">
                Scale Up Threshold:{' '}
                {machineConfig.autoscaling.utilizationScaleUpThreshold * 100}%
              </p>
              <p className="text-sm text-muted-foreground">
                Scale Down Threshold:{' '}
                {machineConfig.autoscaling.utilizationScaleDownThreshold * 100}%
              </p>
              <p className="text-sm text-muted-foreground">
                Scaling Increment: {machineConfig.autoscaling.increment}
              </p>
            </>
          ) : (
            <p className="text-sm text-muted-foreground">
              Static: {machineConfig.numReplicas} replicas
            </p>
          )}
        </div>

        <div className="space-y-2">
          <h3 className="font-medium">Environment Variables</h3>
          <p className="text-sm text-muted-foreground">
            {secrets.add?.length || 0} new variables
          </p>
          <p className="text-sm text-muted-foreground">
            {secrets.update?.length || 0} updated variables
          </p>
          <p className="text-sm text-muted-foreground">
            {secrets.delete?.length || 0} deleted variables
          </p>
        </div>

        <div className="pt-4">
          <Button className="w-full" onClick={onDeploy} disabled={isDeploying}>
            {isDeploying ? 'Deploying...' : 'Deploy Service'}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
