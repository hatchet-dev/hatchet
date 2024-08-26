import { queries } from '@/lib/api';
import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { useQuery } from '@tanstack/react-query';
import { ExclamationTriangleIcon, PlusIcon } from '@heroicons/react/24/outline';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { z } from 'zod';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Input } from '@/components/ui/input';
import { Step, Steps } from '@/components/ui/steps';
import EnvGroupArray, { KeyValueType } from '@/components/ui/envvar';

export const machineTypes = [
  {
    title: '1 CPU, 1 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 1,
    memoryMb: 1024,
  },
  {
    title: '1 CPU, 2 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 1,
    memoryMb: 2048,
  },
  {
    title: '2 CPU, 2 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 2,
    memoryMb: 2048,
  },
  {
    title: '2 CPU, 4 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 2,
    memoryMb: 4096,
  },
  {
    title: '4 CPU, 8 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 4,
    memoryMb: 8192,
  },
  {
    title: '8 CPU, 16 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 8,
    memoryMb: 16384,
  },
  {
    title: '1 CPU, 1 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 1,
    memoryMb: 1024,
  },
  {
    title: '1 CPU, 2 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 1,
    memoryMb: 2048,
  },
  {
    title: '2 CPU, 2 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 2,
    memoryMb: 2048,
  },
  {
    title: '2 CPU, 4 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 2,
    memoryMb: 4096,
  },
  {
    title: '4 CPU, 8 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 4,
    memoryMb: 8192,
  },
  {
    title: '8 CPU, 16 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 8,
    memoryMb: 16384,
  },
];

export const regions = [
  {
    name: 'Amsterdam, Netherlands',
    value: 'ams',
  },
  {
    name: 'Stockholm, Sweden',
    value: 'arn',
  },
  {
    name: 'Atlanta, Georgia (US)',
    value: 'atl',
  },
  {
    name: 'Bogotá, Colombia',
    value: 'bog',
  },
  {
    name: 'Boston, Massachusetts (US)',
    value: 'bos',
  },
  {
    name: 'Paris, France',
    value: 'cdg',
  },
  {
    name: 'Denver, Colorado (US)',
    value: 'den',
  },
  {
    name: 'Dallas, Texas (US)',
    value: 'dfw',
  },
  {
    name: 'Secaucus, NJ (US)',
    value: 'ewr',
  },
  {
    name: 'Ezeiza, Argentina',
    value: 'eze',
  },
  {
    name: 'Guadalajara, Mexico',
    value: 'gdl',
  },
  {
    name: 'Rio de Janeiro, Brazil',
    value: 'gig',
  },
  {
    name: 'Sao Paulo, Brazil',
    value: 'gru',
  },
  {
    name: 'Hong Kong, Hong Kong',
    value: 'hkg',
  },
  {
    name: 'Ashburn, Virginia (US)',
    value: 'iad',
  },
  {
    name: 'Johannesburg, South Africa',
    value: 'jnb',
  },
  {
    name: 'Los Angeles, California (US)',
    value: 'lax',
  },
  {
    name: 'London, United Kingdom',
    value: 'lhr',
  },
  {
    name: 'Madrid, Spain',
    value: 'mad',
  },
  {
    name: 'Miami, Florida (US)',
    value: 'mia',
  },
  {
    name: 'Tokyo, Japan',
    value: 'nrt',
  },
  {
    name: 'Chicago, Illinois (US)',
    value: 'ord',
  },
  {
    name: 'Bucharest, Romania',
    value: 'otp',
  },
  {
    name: 'Phoenix, Arizona (US)',
    value: 'phx',
  },
  {
    name: 'Querétaro, Mexico',
    value: 'qro',
  },
  {
    name: 'Santiago, Chile',
    value: 'scl',
  },
  {
    name: 'Seattle, Washington (US)',
    value: 'sea',
  },
  {
    name: 'Singapore, Singapore',
    value: 'sin',
  },
  {
    name: 'San Jose, California (US)',
    value: 'sjc',
  },
  {
    name: 'Sydney, Australia',
    value: 'syd',
  },
  {
    name: 'Warsaw, Poland',
    value: 'waw',
  },
  {
    name: 'Montreal, Canada',
    value: 'yul',
  },
  {
    name: 'Toronto, Canada',
    value: 'yyz',
  },
];

export const createOrUpdateManagedWorkerSchema = z.object({
  name: z.string(),
  buildConfig: z.object({
    githubInstallationId: z.string().uuid().length(36),
    githubRepositoryOwner: z.string(),
    githubRepositoryName: z.string(),
    githubRepositoryBranch: z.string(),
    steps: z.array(
      z.object({
        buildDir: z.string(),
        dockerfilePath: z.string(),
      }),
    ),
  }),
  runtimeConfig: z.object({
    numReplicas: z.number().min(0).max(16),
    envVars: z.record(z.string()),
    cpuKind: z.string(),
    cpus: z.number(),
    memoryMb: z.number(),
    region: z.string().optional(),
  }),
});

interface CreateWorkerFormProps {
  onSubmit: (opts: z.infer<typeof createOrUpdateManagedWorkerSchema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export default function CreateWorkerForm({
  onSubmit,
  isLoading,
  fieldErrors,
}: CreateWorkerFormProps) {
  const {
    watch,
    handleSubmit,
    control,
    setValue,
    formState: { errors },
  } = useForm<z.infer<typeof createOrUpdateManagedWorkerSchema>>({
    resolver: zodResolver(createOrUpdateManagedWorkerSchema),
    defaultValues: {
      buildConfig: {
        steps: [
          {
            buildDir: '.',
            dockerfilePath: './Dockerfile',
          },
        ],
      },
      runtimeConfig: {
        numReplicas: 1,
        cpuKind: 'shared',
        cpus: 1,
        memoryMb: 1024,
        envVars: {},
        region: 'sea',
      },
    },
  });

  const [machineType, setMachineType] = useState<string>(
    '1 CPU, 1 GB RAM (shared CPU)',
  );

  const region = watch('runtimeConfig.region');
  const installation = watch('buildConfig.githubInstallationId');
  const repoOwner = watch('buildConfig.githubRepositoryOwner');
  const repoName = watch('buildConfig.githubRepositoryName');
  const repoOwnerName = getRepoOwnerName(repoOwner, repoName);
  const branch = watch('buildConfig.githubRepositoryBranch');

  const listInstallationsQuery = useQuery({
    ...queries.github.listInstallations,
  });

  const listReposQuery = useQuery({
    ...queries.github.listRepos(installation),
  });

  const listBranchesQuery = useQuery({
    ...queries.github.listBranches(installation, repoOwner, repoName),
  });

  const [envVars, setEnvVars] = useState<KeyValueType[]>([]);

  const nameError = errors.name?.message?.toString() || fieldErrors?.name;
  const buildDirError =
    errors.buildConfig?.steps?.[0]?.buildDir?.message?.toString() ||
    fieldErrors?.buildDir;
  const dockerfilePathError =
    errors.buildConfig?.steps?.[0]?.dockerfilePath?.message?.toString() ||
    fieldErrors?.dockerfilePath;
  const numReplicasError =
    errors.runtimeConfig?.numReplicas?.message?.toString() ||
    fieldErrors?.numReplicas;
  const envVarsError =
    errors.runtimeConfig?.envVars?.message?.toString() || fieldErrors?.envVars;
  const cpuKindError =
    errors.runtimeConfig?.cpuKind?.message?.toString() || fieldErrors?.cpuKind;
  const cpusError =
    errors.runtimeConfig?.cpus?.message?.toString() || fieldErrors?.cpus;
  const memoryMbError =
    errors.runtimeConfig?.memoryMb?.message?.toString() ||
    fieldErrors?.memoryMb;
  const githubInstallationIdError =
    errors.buildConfig?.githubInstallationId?.message?.toString() ||
    fieldErrors?.githubInstallationId;
  const githubRepositoryOwnerError =
    errors.buildConfig?.githubRepositoryOwner?.message?.toString() ||
    fieldErrors?.githubRepositoryOwner;
  const githubRepositoryNameError =
    errors.buildConfig?.githubRepositoryName?.message?.toString() ||
    fieldErrors?.githubRepositoryName;
  const githubRepositoryBranchError =
    errors.buildConfig?.githubRepositoryBranch?.message?.toString() ||
    fieldErrors?.githubRepositoryBranch;

  useEffect(() => {
    if (
      listInstallationsQuery.isSuccess &&
      listInstallationsQuery.data.rows.length > 0 &&
      !installation
    ) {
      setValue(
        'buildConfig.githubInstallationId',
        listInstallationsQuery.data.rows[0].metadata.id,
      );
    }
  }, [listInstallationsQuery, setValue, installation]);

  // if there are no github accounts linked, ask the user to link one
  if (
    listInstallationsQuery.isSuccess &&
    listInstallationsQuery.data.rows.length === 0
  ) {
    return (
      <Alert>
        <ExclamationTriangleIcon className="h-4 w-4" />
        <AlertTitle className="font-semibold">Link a Github account</AlertTitle>
        <AlertDescription>
          You don't have any Github accounts linked. Please{' '}
          <a
            href="/api/v1/cloud/users/github-app/start"
            className="text-indigo-400"
          >
            link a Github account
          </a>{' '}
          first.
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <>
      <div className="text-sm text-muted-foreground">
        Create a new managed worker.
      </div>
      <Steps className="mt-6">
        <Step title="Name">
          <div className="grid gap-4">
            <div className="text-sm text-muted-foreground">
              Give your worker a name.
            </div>
            <Label htmlFor="name">Name</Label>
            <Controller
              control={control}
              name="name"
              render={({ field }) => {
                return (
                  <Input {...field} id="name" placeholder="my-awesome-worker" />
                );
              }}
            />
            {nameError && (
              <div className="text-sm text-red-500">{nameError}</div>
            )}
          </div>
        </Step>
        <Step title="Build configuration">
          <div className="grid gap-4">
            <div className="text-sm text-muted-foreground">
              Configure the Github repository the worker should deploy from.
            </div>

            <div className="max-w-3xl grid gap-4">
              <Label htmlFor="role">Github account</Label>
              <Controller
                control={control}
                name="buildConfig.githubInstallationId"
                render={({ field }) => {
                  return (
                    <Select
                      onValueChange={(value) => {
                        field.onChange(value);
                        setValue('buildConfig.githubRepositoryOwner', '');
                        setValue('buildConfig.githubRepositoryName', '');
                        setValue('buildConfig.githubRepositoryBranch', '');
                      }}
                    >
                      <div className="text-sm text-muted-foreground">
                        Not seeing your repository?{' '}
                        <a
                          href="/api/v1/cloud/users/github-app/start"
                          className="text-indigo-400"
                        >
                          Link a new repository
                        </a>
                      </div>
                      <SelectTrigger className="w-fit">
                        <SelectValue id="role" placeholder="Choose account" />
                      </SelectTrigger>
                      <SelectContent>
                        {listInstallationsQuery.data?.rows.map((i) => (
                          <SelectItem key={i.metadata.id} value={i.metadata.id}>
                            {i.account_name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  );
                }}
              />
              {githubInstallationIdError && (
                <div className="text-sm text-red-500">
                  {githubInstallationIdError}
                </div>
              )}

              <Label htmlFor="role">Github repo</Label>
              <Controller
                control={control}
                name="buildConfig.githubRepositoryName"
                render={({ field }) => {
                  return (
                    <Select
                      {...field}
                      value={repoOwnerName}
                      onValueChange={(value) => {
                        // get the correct repository id from the repo owner name
                        setValue(
                          'buildConfig.githubRepositoryOwner',
                          getRepoOwner(value) || '',
                        );
                        setValue(
                          'buildConfig.githubRepositoryName',
                          getRepoName(value) || '',
                        );
                        setValue('buildConfig.githubRepositoryBranch', '');
                      }}
                    >
                      <SelectTrigger className="w-fit">
                        <SelectValue
                          id="role"
                          placeholder="Choose repository"
                        />
                      </SelectTrigger>
                      <SelectContent>
                        {listReposQuery.data?.map((i) => (
                          <SelectItem
                            key={i.repo_owner + i.repo_name}
                            value={
                              getRepoOwnerName(i.repo_owner, i.repo_name) || ''
                            }
                          >
                            {i.repo_owner}/{i.repo_name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  );
                }}
              />
              {githubRepositoryOwnerError && (
                <div className="text-sm text-red-500">
                  {githubRepositoryOwnerError}
                </div>
              )}
              {githubRepositoryNameError && (
                <div className="text-sm text-red-500">
                  {githubRepositoryNameError}
                </div>
              )}
              <Label htmlFor="role">Github branch</Label>
              <Controller
                control={control}
                name="buildConfig.githubRepositoryBranch"
                render={({ field }) => {
                  return (
                    <Select onValueChange={field.onChange} {...field}>
                      <SelectTrigger className="w-fit">
                        <SelectValue id="role" placeholder="Choose branch" />
                      </SelectTrigger>
                      <SelectContent>
                        {listBranchesQuery.data?.map((i) => (
                          <SelectItem key={i.branch_name} value={i.branch_name}>
                            {i.branch_name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  );
                }}
              />
              {githubRepositoryBranchError && (
                <div className="text-sm text-red-500">
                  {githubRepositoryBranchError}
                </div>
              )}
              <Label htmlFor="buildDir">Build directory</Label>
              <Controller
                control={control}
                name="buildConfig.steps.0.buildDir"
                render={({ field }) => {
                  return (
                    <Input
                      {...field}
                      placeholder="build"
                      id="buildDir"
                      defaultValue="."
                      disabled={!branch}
                    />
                  );
                }}
              />
              {buildDirError && (
                <div className="text-sm text-red-500">{buildDirError}</div>
              )}
              <Label htmlFor="dockerfile">Path to Dockerfile</Label>
              <Controller
                control={control}
                name="buildConfig.steps.0.dockerfilePath"
                render={({ field }) => {
                  return (
                    <Input
                      {...field}
                      placeholder="."
                      id="dockerfile"
                      defaultValue="./Dockerfile"
                      disabled={!branch}
                    />
                  );
                }}
              />
              {dockerfilePathError && (
                <div className="text-sm text-red-500">
                  {dockerfilePathError}
                </div>
              )}
            </div>
          </div>
        </Step>
        <Step title="Runtime configuration">
          <div className="grid gap-4">
            <div className="text-sm text-muted-foreground">
              Configure the runtime settings for this worker.
            </div>
            <Label htmlFor="region">Region</Label>
            <Select
              value={region}
              onValueChange={(value) => {
                setValue('runtimeConfig.region', value);
              }}
            >
              <SelectTrigger className="w-fit">
                <SelectValue id="region" placeholder="Choose region" />
              </SelectTrigger>
              <SelectContent>
                {regions.map((i) => (
                  <SelectItem key={i.value} value={i.value}>
                    {i.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Label htmlFor="numReplicas">Number of replicas</Label>
            <Controller
              control={control}
              name="runtimeConfig.numReplicas"
              render={({ field }) => {
                return (
                  <Input
                    {...field}
                    type="number"
                    onChange={(e) => {
                      if (e.target.value === '') {
                        field.onChange(e.target.value);
                        return;
                      }

                      field.onChange(parseInt(e.target.value));
                    }}
                    min={0}
                    max={16}
                    id="numReplicas"
                    placeholder="1"
                  />
                );
              }}
            />
            {numReplicasError && (
              <div className="text-sm text-red-500">{numReplicasError}</div>
            )}
            <Label htmlFor="machineType">Machine type</Label>
            <Controller
              control={control}
              name="runtimeConfig.cpuKind"
              render={({ field }) => {
                return (
                  <Select
                    {...field}
                    value={machineType}
                    onValueChange={(value) => {
                      // get the correct machine type from the value
                      const machineType = machineTypes.find(
                        (i) => i.title === value,
                      );

                      setMachineType(value);
                      setValue('runtimeConfig.cpus', machineType?.cpus || 1);
                      setValue(
                        'runtimeConfig.memoryMb',
                        machineType?.memoryMb || 1024,
                      );
                      setValue(
                        'runtimeConfig.cpuKind',
                        machineType?.cpuKind || 'shared',
                      );
                    }}
                  >
                    <SelectTrigger className="w-fit">
                      <SelectValue id="machineType" placeholder="Choose type" />
                    </SelectTrigger>
                    <SelectContent>
                      {machineTypes.map((i) => (
                        <SelectItem key={i.title} value={i.title}>
                          {i.title}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                );
              }}
            />
            {cpuKindError && (
              <div className="text-sm text-red-500">{cpuKindError}</div>
            )}
            {cpusError && (
              <div className="text-sm text-red-500">{cpusError}</div>
            )}
            {memoryMbError && (
              <div className="text-sm text-red-500">{memoryMbError}</div>
            )}
            <Label>Environment Variables</Label>
            <EnvGroupArray
              values={envVars}
              setValues={(value) => {
                setEnvVars(value);
                setValue(
                  'runtimeConfig.envVars',
                  value.reduce<Record<string, string>>((acc, item) => {
                    acc[item.key] = item.value;
                    return acc;
                  }, {}),
                );
              }}
            />
            {envVarsError && (
              <div className="text-sm text-red-500">{envVarsError}</div>
            )}
          </div>
        </Step>
        <Step title="Review">
          <div className="grid gap-4">
            <div className="text-sm text-muted-foreground">
              Review the settings for this worker.
            </div>
            <Button
              onClick={handleSubmit(onSubmit)}
              disabled={!installation || !repoOwnerName || !branch}
              className="w-fit px-8"
            >
              {isLoading && <PlusIcon className="h-4 w-4 animate-spin" />}
              Create worker
            </Button>
          </div>
        </Step>
      </Steps>
    </>
  );
}

export function getRepoOwnerName(repoOwner: string, repoName: string) {
  if (!repoOwner || !repoName) {
    return;
  }
  return `${repoOwner}::${repoName}`;
}

export function getRepoOwner(repoOwnerName?: string) {
  if (!repoOwnerName) {
    return;
  }

  const splArr = repoOwnerName.split('::');
  if (splArr.length > 1) {
    return splArr[0];
  }
}

export function getRepoName(repoOwnerName?: string) {
  if (!repoOwnerName) {
    return;
  }

  const splArr = repoOwnerName.split('::');
  if (splArr.length > 1) {
    return splArr[1];
  }
}
