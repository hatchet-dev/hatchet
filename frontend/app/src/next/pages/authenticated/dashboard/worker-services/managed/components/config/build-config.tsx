import { Label } from '@/next/components/ui/label';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/next/components/ui/card';
import { Input } from '@/next/components/ui/input';
import { CreateBuildStepRequest } from '@/lib/api/generated/cloud/data-contracts';
import { GithubRepoSelectorValue } from './github-repo-selector';
import { useEffect, useState } from 'react';

const sanitizeServiceName = (name: string): string => {
  return name.replace(/[^a-zA-Z0-9-]/g, '-');
};

export type BuildConfigValue = CreateBuildStepRequest & {
  serviceName: string;
};

interface BuildConfigProps {
  githubRepo: GithubRepoSelectorValue;
  value: BuildConfigValue;
  onChange: (value: BuildConfigValue) => void;
  type: 'update' | 'create';
  actions?: React.ReactNode;
}

export function BuildConfig({
  githubRepo,
  value,
  onChange,
  type,
  actions,
}: BuildConfigProps) {
  const [isNamePristine, setIsNamePristine] = useState(true);

  useEffect(() => {
    if (
      !isNamePristine ||
      githubRepo.githubRepositoryName === '' ||
      type === 'update'
    ) {
      return;
    }

    const dockerfileName = value.dockerfilePath.split('/').pop();
    const dockerfileServiceName = dockerfileName
      ?.split('.')
      .filter((part) => part !== 'Dockerfile')
      .pop();

    onChange({
      ...value,
      serviceName: sanitizeServiceName(
        [
          githubRepo.githubRepositoryName,
          githubRepo.githubRepositoryBranch,
          dockerfileServiceName,
        ]
          .filter(Boolean)
          .join('-')
          .toLowerCase(),
      ),
    });
  }, [githubRepo, onChange, value, isNamePristine, value.dockerfilePath]);

  return (
    <Card variant={type === 'update' ? 'borderless' : 'default'}>
      <CardHeader>
        <CardTitle>Build Configuration</CardTitle>
        <CardDescription>
          Configure the Docker build settings for your worker service.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="build-dir">Build Directory</Label>
          <Input
            id="build-dir"
            placeholder="e.g. ./"
            value={value.buildDir || ''}
            onChange={(e) => {
              onChange({
                ...value,
                buildDir: e.target.value,
              });
            }}
          />
          <p className="text-sm text-muted-foreground">
            The relative path to the build directory
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="dockerfile-path">Dockerfile Path</Label>
          <Input
            id="dockerfile-path"
            placeholder="e.g. ./Dockerfile"
            value={value.dockerfilePath || ''}
            onChange={(e) => {
              onChange({
                ...value,
                dockerfilePath: e.target.value,
              });
            }}
          />
          <p className="text-sm text-muted-foreground">
            The relative path from the build directory to the Dockerfile
          </p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="service-name">Service Name</Label>
          <Input
            id="service-name"
            placeholder="e.g. my-service"
            value={value.serviceName || ''}
            onChange={(e) => {
              setIsNamePristine(false);
              onChange({
                ...value,
                serviceName: sanitizeServiceName(e.target.value),
              });
            }}
          />
          <p className="text-sm text-muted-foreground">
            A friendly name for the service
          </p>
        </div>
      </CardContent>
      {actions}
    </Card>
  );
}
