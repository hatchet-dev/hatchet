import { queries } from '@/lib/api';
import { useEffect, useState } from 'react';
import { Button } from '@/components/v1/ui/button';
import { useQuery } from '@tanstack/react-query';
import { ExclamationTriangleIcon, PlusIcon } from '@heroicons/react/24/outline';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { z } from 'zod';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Label } from '@/components/v1/ui/label';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Input } from '@/components/v1/ui/input';
import EnvGroupArray, { KeyValueType } from '@/components/v1/ui/envvar';
import {
  getRepoName,
  getRepoOwner,
  getRepoOwnerName,
  machineTypes,
  regions,
  ScalingType,
  scalingTypes,
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
} from '@/components/v1/ui/accordion';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { Checkbox } from '@/components/v1/ui/checkbox';

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
      numReplicas: z.number().min(0).max(16).optional(),
      cpuKind: z.string(),
      cpus: z.number(),
      memoryMb: z.number(),
      regions: z.array(z.nativeEnum(ManagedWorkerRegion)).optional(),
      autoscaling: z
        .object({
          waitDuration: z.string(),
          rollingWindowDuration: z.string(),
          utilizationScaleUpThreshold: z.number(),
          utilizationScaleDownThreshold: z.number(),
          increment: z.number(),
          minAwakeReplicas: z.number(),
          maxReplicas: z.number(),
          scaleToZero: z.boolean(),
          fly: z.object({
            autoscalingKey: z.string(),
            currentReplicas: z.number(),
          }),
        })
        .optional(),
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
              numReplicas:
                managedWorker.runtimeConfigs[0].autoscaling != undefined
                  ? undefined
                  : managedWorker.runtimeConfigs[0].numReplicas,
              cpuKind: managedWorker.runtimeConfigs[0].cpuKind,
              cpus: managedWorker.runtimeConfigs[0].cpus,
              memoryMb: managedWorker.runtimeConfigs[0].memoryMb,
              regions: [managedWorker.runtimeConfigs[0].region],
              autoscaling:
                managedWorker.runtimeConfigs[0].autoscaling != undefined
                  ? {
                      waitDuration:
                        managedWorker.runtimeConfigs[0].autoscaling
                          .waitDuration,
                      rollingWindowDuration:
                        managedWorker.runtimeConfigs[0].autoscaling
                          .rollingWindowDuration,
                      utilizationScaleUpThreshold:
                        managedWorker.runtimeConfigs[0].autoscaling
                          .utilizationScaleUpThreshold,
                      utilizationScaleDownThreshold:
                        managedWorker.runtimeConfigs[0].autoscaling
                          .utilizationScaleDownThreshold,
                      increment:
                        managedWorker.runtimeConfigs[0].autoscaling.increment,
                      minAwakeReplicas:
                        managedWorker.runtimeConfigs[0].autoscaling
                          .minAwakeReplicas,
                      maxReplicas:
                        managedWorker.runtimeConfigs[0].autoscaling.maxReplicas,
                      scaleToZero:
                        managedWorker.runtimeConfigs[0].autoscaling.scaleToZero,
                      fly: {
                        autoscalingKey: 'dashboard',
                        currentReplicas:
                          managedWorker.runtimeConfigs[0].autoscaling
                            .minAwakeReplicas,
                      },
                    }
                  : undefined,
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
  const [scalingType, setScalingType] = useState<ScalingType>(
    managedWorker.runtimeConfigs?.length == 1 &&
      managedWorker.runtimeConfigs[0].autoscaling != undefined
      ? 'Autoscaling'
      : 'Static',
  );

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
  const autoscalingWaitDurationError =
    errors.runtimeConfig?.autoscaling?.waitDuration?.message?.toString() ||
    fieldErrors?.waitDuration;
  const autoscalingRollingWindowDurationError =
    errors.runtimeConfig?.autoscaling?.rollingWindowDuration?.message?.toString() ||
    fieldErrors?.rollingWindowDuration;
  const autoscalingUtilizationScaleUpThresholdError =
    errors.runtimeConfig?.autoscaling?.utilizationScaleUpThreshold?.message?.toString() ||
    fieldErrors?.utilizationScaleUpThreshold;
  const autoscalingUtilizationScaleDownThresholdError =
    errors.runtimeConfig?.autoscaling?.utilizationScaleDownThreshold?.message?.toString() ||
    fieldErrors?.utilizationScaleDownThreshold;
  const autoscalingIncrementError =
    errors.runtimeConfig?.autoscaling?.increment?.message?.toString() ||
    fieldErrors?.increment;
  const autoscalingMinAwakeReplicasError =
    errors.runtimeConfig?.autoscaling?.minAwakeReplicas?.message?.toString() ||
    fieldErrors?.minAwakeReplicas;
  const autoscalingMaxReplicasError =
    errors.runtimeConfig?.autoscaling?.maxReplicas?.message?.toString() ||
    fieldErrors?.maxReplicas;
  const autoscalingScaleToZeroError =
    errors.runtimeConfig?.autoscaling?.scaleToZero?.message?.toString() ||
    fieldErrors?.scaleToZero;

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
                  <Label>Scaling Method</Label>
                  <Tabs
                    defaultValue="Static"
                    value={scalingType}
                    onValueChange={(value) => {
                      if (value === 'Static') {
                        setScalingType('Static');
                        setValue('runtimeConfig.numReplicas', 1);
                        setValue('runtimeConfig.autoscaling', undefined);
                        return;
                      } else {
                        setScalingType('Autoscaling');
                        setValue('runtimeConfig.numReplicas', undefined);
                        setValue('runtimeConfig.autoscaling', {
                          waitDuration: '1m',
                          rollingWindowDuration: '2m',
                          utilizationScaleUpThreshold: 0.75,
                          utilizationScaleDownThreshold: 0.25,
                          increment: 1,
                          scaleToZero: true,
                          minAwakeReplicas: 1,
                          maxReplicas: 10,
                          fly: {
                            autoscalingKey: 'dashboard',
                            currentReplicas: 1,
                          },
                        });
                      }
                    }}
                  >
                    <TabsList layout="underlined">
                      {scalingTypes.map((type) => (
                        <TabsTrigger
                          variant="underlined"
                          value={type}
                          key={type}
                        >
                          {type}
                        </TabsTrigger>
                      ))}
                    </TabsList>
                    <TabsContent value="Static" className="pt-4 grid gap-4">
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
                        <div className="text-sm text-red-500">
                          {numReplicasError}
                        </div>
                      )}
                    </TabsContent>
                    <TabsContent
                      value="Autoscaling"
                      className="pt-4 grid gap-4"
                    >
                      <Label htmlFor="minAwakeReplicas">Min Replicas</Label>
                      <Controller
                        control={control}
                        name="runtimeConfig.autoscaling.minAwakeReplicas"
                        render={({ field }) => {
                          return (
                            <Input
                              {...field}
                              id="minAwakeReplicas"
                              type="number"
                              onChange={(e) => {
                                field.onChange(parseInt(e.target.value));

                                setValue(
                                  'runtimeConfig.autoscaling.fly.currentReplicas',
                                  parseInt(e.target.value),
                                );
                              }}
                            />
                          );
                        }}
                      />
                      {autoscalingMinAwakeReplicasError && (
                        <div className="text-sm text-red-500">
                          {autoscalingMinAwakeReplicasError}
                        </div>
                      )}
                      <Label htmlFor="maxReplicas">Max Replicas</Label>
                      <Controller
                        control={control}
                        name="runtimeConfig.autoscaling.maxReplicas"
                        render={({ field }) => {
                          return (
                            <Input
                              {...field}
                              id="maxReplicas"
                              type="number"
                              onChange={(e) => {
                                field.onChange(parseInt(e.target.value));
                              }}
                            />
                          );
                        }}
                      />
                      {autoscalingMaxReplicasError && (
                        <div className="text-sm text-red-500">
                          {autoscalingMaxReplicasError}
                        </div>
                      )}
                      <Controller
                        control={control}
                        name="runtimeConfig.autoscaling.scaleToZero"
                        render={({ field }) => {
                          return (
                            <div className="flex flex-row gap-4 items-center">
                              <Label htmlFor="scaleToZero">
                                Scale to zero during periods of inactivity?
                              </Label>
                              <Checkbox
                                checked={field.value}
                                onCheckedChange={field.onChange}
                              />
                            </div>
                          );
                        }}
                      />
                      {autoscalingScaleToZeroError && (
                        <div className="text-sm text-red-500">
                          {autoscalingScaleToZeroError}
                        </div>
                      )}
                      <Accordion type="single" collapsible>
                        <AccordionItem value="advanced">
                          <AccordionTrigger>
                            Advanced autoscaling settings
                          </AccordionTrigger>
                          <AccordionContent className="flex flex-col gap-4">
                            <Label htmlFor="waitDuration">Wait Duration</Label>
                            <div className="text-sm text-muted-foreground">
                              How long to wait between autoscaling events. For
                              example: 10s (10 seconds), 5m (5 minutes), 1h (1
                              hour).
                            </div>
                            <Controller
                              control={control}
                              name="runtimeConfig.autoscaling.waitDuration"
                              render={({ field }) => {
                                return (
                                  <Input
                                    {...field}
                                    id="waitDuration"
                                    placeholder="1m"
                                  />
                                );
                              }}
                            />
                            {autoscalingWaitDurationError && (
                              <div className="text-sm text-red-500">
                                {autoscalingWaitDurationError}
                              </div>
                            )}
                            <Label htmlFor="rollingWindowDuration">
                              Rolling Window Duration
                            </Label>
                            <div className="text-sm text-muted-foreground">
                              The amount of time to look at utilization metrics
                              for autoscaling. Lower values will lead to faster
                              scale-up and scale-down. Example: 2m (2 minutes),
                              5m (5 minutes), 1h (1 hour).
                            </div>
                            <Controller
                              control={control}
                              name="runtimeConfig.autoscaling.rollingWindowDuration"
                              render={({ field }) => {
                                return (
                                  <Input
                                    {...field}
                                    id="rollingWindowDuration"
                                    placeholder="2m"
                                  />
                                );
                              }}
                            />
                            {autoscalingRollingWindowDurationError && (
                              <div className="text-sm text-red-500">
                                {autoscalingRollingWindowDurationError}
                              </div>
                            )}
                            <Label htmlFor="utilizationScaleUpThreshold">
                              Utilization Scale Up Threshold
                            </Label>
                            <div className="text-sm text-muted-foreground">
                              A value between 0 and 1 which represents the
                              utilization threshold at which to scale up. For
                              example, 0.75 means that if the utilization is
                              above 75%, scale up.
                            </div>
                            <Controller
                              control={control}
                              name="runtimeConfig.autoscaling.utilizationScaleUpThreshold"
                              render={({ field }) => {
                                return (
                                  <Input
                                    {...field}
                                    id="utilizationScaleUpThreshold"
                                    type="number"
                                    min={0}
                                    max={1}
                                    step={0.01}
                                    onChange={(e) => {
                                      field.onChange(
                                        parseFloat(e.target.value),
                                      );
                                    }}
                                  />
                                );
                              }}
                            />
                            {autoscalingUtilizationScaleUpThresholdError && (
                              <div className="text-sm text-red-500">
                                {autoscalingUtilizationScaleUpThresholdError}
                              </div>
                            )}
                            <Label htmlFor="utilizationScaleDownThreshold">
                              Utilization Scale Down Threshold
                            </Label>
                            <div className="text-sm text-muted-foreground">
                              A value between 0 and 1 which represents the
                              utilization threshold at which to scale down. For
                              example, 0.25 means that if the utilization is
                              below 25%, scale down.
                            </div>
                            <Controller
                              control={control}
                              name="runtimeConfig.autoscaling.utilizationScaleDownThreshold"
                              render={({ field }) => {
                                return (
                                  <Input
                                    {...field}
                                    id="utilizationScaleDownThreshold"
                                    type="number"
                                    min={0}
                                    max={1}
                                    step={0.01}
                                    onChange={(e) => {
                                      field.onChange(
                                        parseFloat(e.target.value),
                                      );
                                    }}
                                  />
                                );
                              }}
                            />
                            {autoscalingUtilizationScaleDownThresholdError && (
                              <div className="text-sm text-red-500">
                                {autoscalingUtilizationScaleDownThresholdError}
                              </div>
                            )}
                            <Label htmlFor="increment">Scaling Increment</Label>
                            <div className="text-sm text-muted-foreground">
                              The number of replicas to scale by when scaling up
                              or down.
                            </div>
                            <Controller
                              control={control}
                              name="runtimeConfig.autoscaling.increment"
                              render={({ field }) => {
                                return (
                                  <Input
                                    {...field}
                                    id="increment"
                                    type="number"
                                    onChange={(e) => {
                                      field.onChange(parseInt(e.target.value));
                                    }}
                                  />
                                );
                              }}
                            />
                            {autoscalingIncrementError && (
                              <div className="text-sm text-red-500">
                                {autoscalingIncrementError}
                              </div>
                            )}
                          </AccordionContent>
                        </AccordionItem>
                      </Accordion>
                    </TabsContent>
                  </Tabs>
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
