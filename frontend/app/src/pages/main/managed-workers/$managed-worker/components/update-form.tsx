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
import EnvGroupArray, { KeyValueType } from '@/components/ui/envvar';
import {
  getRepoName,
  getRepoOwner,
  getRepoOwnerName,
  machineTypes,
  regions,
} from '../../create/components/create-worker-form';
import {
  ManagedWorker,
  ManagedWorkerRegion,
} from '@/lib/api/generated/cloud/data-contracts';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

interface UpdateWorkerFormProps {
  onSubmit: (opts: z.infer<typeof updateManagedWorkerSchema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
  managedWorker: ManagedWorker;
}

const updateManagedWorkerSchema = z.object({
  name: z.string().optional(),
  buildConfig: z
    .object({
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
    })
    .optional(),
  isIac: z.boolean().default(false).optional(),
  envVars: z.record(z.string()).optional(),
  runtimeConfig: z
    .object({
      numReplicas: z.number().min(0).max(16),
      cpuKind: z.string(),
      cpus: z.number(),
      memoryMb: z.number(),
      regions: z.array(z.nativeEnum(ManagedWorkerRegion)).optional(),
    })
    .optional(),
});

export default function UpdateWorkerForm({
  onSubmit,
  isLoading,
  fieldErrors,
  managedWorker,
}: UpdateWorkerFormProps) {
  const {
    watch,
    handleSubmit,
    control,
    setValue,
    getValues,
    formState: { errors },
  } = useForm<z.infer<typeof updateManagedWorkerSchema>>({
    resolver: zodResolver(updateManagedWorkerSchema),
    defaultValues: {
      name: managedWorker.name,
      buildConfig: {
        githubInstallationId: managedWorker.buildConfig.githubInstallationId,
        githubRepositoryBranch:
          managedWorker.buildConfig.githubRepositoryBranch,
        githubRepositoryName:
          managedWorker.buildConfig.githubRepository.repo_name,
        githubRepositoryOwner:
          managedWorker.buildConfig.githubRepository.repo_owner,
        steps: managedWorker.buildConfig.steps?.map((step) => ({
          buildDir: step.buildDir,
          dockerfilePath: step.dockerfilePath,
        })) || [
          {
            buildDir: '.',
            dockerfilePath: './Dockerfile',
          },
        ],
      },
      envVars: managedWorker.envVars,
      isIac: managedWorker.isIac,
      runtimeConfig:
        !managedWorker.isIac && managedWorker.runtimeConfigs?.length == 1
          ? {
              numReplicas: managedWorker.runtimeConfigs[0].numReplicas,
              cpuKind: managedWorker.runtimeConfigs[0].cpuKind,
              cpus: managedWorker.runtimeConfigs[0].cpus,
              memoryMb: managedWorker.runtimeConfigs[0].memoryMb,
              regions: [managedWorker.runtimeConfigs[0].region],
            }
          : undefined,
    },
  });

  const [machineType, setMachineType] = useState<string>(
    '1 CPU, 1 GB RAM (shared CPU)',
  );

  const region = watch('runtimeConfig.regions');
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

  const [envVars, setEnvVars] = useState<KeyValueType[]>(
    envVarsRecordToKeyValueType(managedWorker.envVars),
  );

  const [isIac, setIsIac] = useState(managedWorker.isIac);

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
    errors.envVars?.message?.toString() || fieldErrors?.envVars;
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
        managedWorker.buildConfig.githubInstallationId ||
          listInstallationsQuery.data.rows[0].metadata.id,
      );
    }
  }, [managedWorker, listInstallationsQuery, setValue, installation]);

  useEffect(() => {
    if (!isIac && !getValues('runtimeConfig')) {
      setValue('runtimeConfig', {
        numReplicas: 1,
        cpuKind: 'shared',
        cpus: 1,
        memoryMb: 1024,
        regions: [ManagedWorkerRegion.Sea],
      });
      setMachineType('1 CPU, 1 GB RAM (shared CPU)');
      setValue('runtimeConfig.regions', [ManagedWorkerRegion.Sea]);
    }
  }, [getValues, setValue, isIac]);

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
        Change the configuration of your worker. This will trigger a
        redeployment.
      </div>
      <div className="mt-6 flex flex-col gap-4">
        <div>
          <div className="grid gap-4">
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
        </div>
        <Accordion type="multiple" className="w-full">
          <AccordionItem value="build">
            <AccordionTrigger className="text-lg font-semibold text-foreground">
              Build configuration
            </AccordionTrigger>
            <AccordionContent className="max-w-3xl grid gap-4">
              <Label htmlFor="role">Github account</Label>
              <Controller
                control={control}
                name="buildConfig.githubInstallationId"
                render={({ field }) => {
                  return (
                    <Select
                      value={installation}
                      onValueChange={(value) => {
                        field.onChange(value);
                        setValue('buildConfig.githubRepositoryOwner', '');
                        setValue('buildConfig.githubRepositoryName', '');
                        setValue('buildConfig.githubRepositoryBranch', '');
                      }}
                    >
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
            </AccordionContent>
          </AccordionItem>
          <AccordionItem value="runtime">
            <AccordionTrigger className="text-lg font-semibold text-foreground">
              Runtime configuration
            </AccordionTrigger>
            <AccordionContent className="grid gap-4">
              <div className="text-sm text-muted-foreground">
                Configure the runtime settings for this worker.
              </div>
              <Label>Environment Variables</Label>
              <EnvGroupArray
                values={envVars}
                setValues={(value) => {
                  setEnvVars(value);
                  setValue(
                    'envVars',
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
              <Label>Machine Configuration Method</Label>
              <Tabs
                defaultValue="activity"
                value={isIac ? 'iac' : 'ui'}
                onValueChange={(value) => {
                  setIsIac(value === 'iac');
                  setValue('isIac', value === 'iac');
                }}
              >
                <TabsList layout="underlined">
                  <TabsTrigger variant="underlined" value="ui">
                    UI
                  </TabsTrigger>
                  <TabsTrigger variant="underlined" value="iac">
                    Infra-As-Code
                  </TabsTrigger>
                </TabsList>
                <TabsContent value="iac" className="pt-4 grid gap-4">
                  <a
                    href="https://docs.hatchet.run/compute/cpu"
                    className="underline"
                  >
                    Learn how to configure infra-as-code.
                  </a>
                </TabsContent>
                <TabsContent value="ui" className="pt-4 grid gap-4">
                  <Label htmlFor="region">Region</Label>
                  <Select
                    value={region?.toString()}
                    onValueChange={(value) => {
                      // find the region object from the value
                      const region = regions.find((i) => i.value === value);

                      if (!region) {
                        return;
                      }

                      setValue('runtimeConfig.regions', [region.value]);
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
                          min={1}
                          max={16}
                          id="numReplicas"
                          placeholder="1"
                        />
                      );
                    }}
                  />
                  {numReplicasError && (
                    <div className="text-sm text-red-500">
                      {numReplicasError}
                    </div>
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
                            setValue(
                              'runtimeConfig.cpus',
                              machineType?.cpus || 1,
                            );
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
                            <SelectValue
                              id="machineType"
                              placeholder="Choose type"
                            />
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
                </TabsContent>
              </Tabs>
            </AccordionContent>
          </AccordionItem>
        </Accordion>
        <Button
          onClick={handleSubmit(onSubmit)}
          disabled={!installation || !repoOwnerName || !branch}
          className="w-fit px-8"
        >
          {isLoading && <PlusIcon className="h-4 w-4 animate-spin" />}
          Save changes
        </Button>
      </div>
    </>
  );
}

function envVarsRecordToKeyValueType(
  envVars: Record<string, string>,
): KeyValueType[] {
  return Object.entries(envVars).map(([key, value]) => ({
    key,
    value,
    hidden: false,
    locked: false,
    deleted: false,
  }));
}
