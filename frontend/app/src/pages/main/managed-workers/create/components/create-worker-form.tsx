import { queries } from '@/lib/api';
import { useEffect, useState, useMemo } from 'react';
import { Button } from '@/components/v1/ui/button';
import { useQuery } from '@tanstack/react-query';
import { PlusIcon, ArrowUpIcon } from '@heroicons/react/24/outline';
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
import { Input } from '@/components/v1/ui/input';
import { Step, Steps } from '@/components/v1/ui/steps';
import EnvGroupArray, { KeyValueType } from '@/components/v1/ui/envvar';
import { ManagedWorkerRegion } from '@/lib/api/generated/cloud/data-contracts';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { Checkbox } from '@/components/v1/ui/checkbox';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/v1/ui/accordion';
import { useTenant } from '@/lib/atoms';
import {
  managedCompute,
  ComputeType,
} from '@/lib/can/features/managed-compute';

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
    value: ManagedWorkerRegion.Ams,
  },
  {
    name: 'Stockholm, Sweden',
    value: ManagedWorkerRegion.Arn,
  },
  {
    name: 'Atlanta, Georgia (US)',
    value: ManagedWorkerRegion.Atl,
  },
  {
    name: 'Bogotá, Colombia',
    value: ManagedWorkerRegion.Bog,
  },
  {
    name: 'Boston, Massachusetts (US)',
    value: ManagedWorkerRegion.Bos,
  },
  {
    name: 'Paris, France',
    value: ManagedWorkerRegion.Cdg,
  },
  {
    name: 'Denver, Colorado (US)',
    value: ManagedWorkerRegion.Den,
  },
  {
    name: 'Dallas, Texas (US)',
    value: ManagedWorkerRegion.Dfw,
  },
  {
    name: 'Secaucus, NJ (US)',
    value: ManagedWorkerRegion.Ewr,
  },
  {
    name: 'Ezeiza, Argentina',
    value: ManagedWorkerRegion.Eze,
  },
  {
    name: 'Guadalajara, Mexico',
    value: ManagedWorkerRegion.Gdl,
  },
  {
    name: 'Rio de Janeiro, Brazil',
    value: ManagedWorkerRegion.Gig,
  },
  {
    name: 'Sao Paulo, Brazil',
    value: ManagedWorkerRegion.Gru,
  },
  {
    name: 'Hong Kong, Hong Kong',
    value: ManagedWorkerRegion.Hkg,
  },
  {
    name: 'Ashburn, Virginia (US)',
    value: ManagedWorkerRegion.Iad,
  },
  {
    name: 'Johannesburg, South Africa',
    value: ManagedWorkerRegion.Jnb,
  },
  {
    name: 'Los Angeles, California (US)',
    value: ManagedWorkerRegion.Lax,
  },
  {
    name: 'London, United Kingdom',
    value: ManagedWorkerRegion.Lhr,
  },
  {
    name: 'Madrid, Spain',
    value: ManagedWorkerRegion.Mad,
  },
  {
    name: 'Miami, Florida (US)',
    value: ManagedWorkerRegion.Mia,
  },
  {
    name: 'Tokyo, Japan',
    value: ManagedWorkerRegion.Nrt,
  },
  {
    name: 'Chicago, Illinois (US)',
    value: ManagedWorkerRegion.Ord,
  },
  {
    name: 'Bucharest, Romania',
    value: ManagedWorkerRegion.Otp,
  },
  {
    name: 'Phoenix, Arizona (US)',
    value: ManagedWorkerRegion.Phx,
  },
  {
    name: 'Querétaro, Mexico',
    value: ManagedWorkerRegion.Qro,
  },
  {
    name: 'Santiago, Chile',
    value: ManagedWorkerRegion.Scl,
  },
  {
    name: 'Seattle, Washington (US)',
    value: ManagedWorkerRegion.Sea,
  },
  {
    name: 'Singapore, Singapore',
    value: ManagedWorkerRegion.Sin,
  },
  {
    name: 'San Jose, California (US)',
    value: ManagedWorkerRegion.Sjc,
  },
  {
    name: 'Sydney, Australia',
    value: ManagedWorkerRegion.Syd,
  },
  {
    name: 'Warsaw, Poland',
    value: ManagedWorkerRegion.Waw,
  },
  {
    name: 'Montreal, Canada',
    value: ManagedWorkerRegion.Yul,
  },
  {
    name: 'Toronto, Canada',
    value: ManagedWorkerRegion.Yyz,
  },
];

export type ScalingType = 'Autoscaling' | 'Static';
export const scalingTypes: ScalingType[] = ['Static', 'Autoscaling'];

const createManagedWorkerSchema = z.object({
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
  isIac: z.boolean().default(false),
  secrets: z.object({
    add: z.array(
      z.object({
        key: z.string(),
        value: z.string(),
      }),
    ),
  }),
  runtimeConfig: z.object({
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
  }),
});

interface CreateWorkerFormProps {
  onSubmit: (opts: z.infer<typeof createManagedWorkerSchema>) => void;
  tenantId: string;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export default function CreateWorkerForm({
  onSubmit,
  tenantId,
  isLoading,
  fieldErrors,
}: CreateWorkerFormProps) {
  const { can } = useTenant();

  const {
    watch,
    handleSubmit,
    control,
    setValue,
    formState: { errors },
  } = useForm<z.infer<typeof createManagedWorkerSchema>>({
    resolver: zodResolver(createManagedWorkerSchema),
    defaultValues: {
      buildConfig: {
        steps: [
          {
            buildDir: '.',
            dockerfilePath: './Dockerfile',
          },
        ],
      },
      secrets: {
        add: [],
      },
      runtimeConfig: {
        numReplicas: 1,
        cpuKind: 'shared',
        cpus: 1,
        memoryMb: 1024,
        regions: [ManagedWorkerRegion.Sea],
      },
    },
  });

  const [machineType, setMachineType] = useState<string>(
    '1 CPU, 1 GB RAM (shared CPU)',
  );

  // Get max replicas based on plan
  const getMaxReplicas = useMemo(() => {
    if (!can) {
      return Infinity;
    }

    // Check maximum replicas for each plan
    for (let i = 1; i <= 20; i++) {
      const [allowed] = can(managedCompute.maxReplicas(i));
      if (!allowed) {
        return i - 1; // Return the last valid replica count
      }
    }
    return 20; // Default to maximum
  }, [can]);

  // Check if the currently selected compute type is available on the user's plan
  const [isComputeAllowed] = useMemo(() => {
    const selectedMachine = machineTypes.find((m) => m.title === machineType);
    if (!selectedMachine || !can) {
      return [true, undefined];
    }

    const computeType: ComputeType = {
      cpuKind: selectedMachine.cpuKind,
      cpus: selectedMachine.cpus,
      memoryMb: selectedMachine.memoryMb,
    };

    return can(managedCompute.selectCompute(computeType));
  }, [can, machineType]);

  // Display a message when a feature requires an upgrade

  const region = watch('runtimeConfig.regions');
  const installation = watch('buildConfig.githubInstallationId');
  const repoOwner = watch('buildConfig.githubRepositoryOwner');
  const repoName = watch('buildConfig.githubRepositoryName');
  const repoOwnerName = getRepoOwnerName(repoOwner, repoName);
  const branch = watch('buildConfig.githubRepositoryBranch');
  const numReplicas = watch('runtimeConfig.numReplicas');
  const autoscalingMaxReplicas = watch('runtimeConfig.autoscaling.maxReplicas');

  const listInstallationsQuery = useQuery({
    ...queries.github.listInstallations(tenantId),
  });

  const listReposQuery = useQuery({
    ...queries.github.listRepos(tenantId, installation),
  });

  const listBranchesQuery = useQuery({
    ...queries.github.listBranches(tenantId, installation, repoOwner, repoName),
  });

  // Check if the current replica count exceeds the plan limit
  const isReplicaCountAllowed = useMemo(() => {
    if (!numReplicas || !can) {
      return true;
    }
    const [allowed] = can(managedCompute.maxReplicas(numReplicas));
    return allowed;
  }, [can, numReplicas]);

  // Check if the autoscaling max replicas exceeds the plan limit
  const isAutoscalingMaxReplicasAllowed = useMemo(() => {
    if (!autoscalingMaxReplicas || !can) {
      return true;
    }
    const [allowed] = can(managedCompute.maxReplicas(autoscalingMaxReplicas));
    return allowed;
  }, [can, autoscalingMaxReplicas]);

  const [secrets, setSecrets] = useState<KeyValueType[]>([]);
  const [isIac, setIsIac] = useState(false);
  const [scalingType, setScalingType] = useState<ScalingType>('Static');

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
  const secretsError =
    errors.secrets?.add?.message?.toString() || fieldErrors?.secrets;
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
        listInstallationsQuery.data.rows[0].metadata.id,
      );
    }
  }, [listInstallationsQuery, setValue, installation]);

  // TODO: Add this back in
  // if (
  //   listInstallationsQuery.isSuccess &&
  //   listInstallationsQuery.data.rows.length === 0
  // ) {
  //   return (
  //     <Alert>
  //       <ExclamationTriangleIcon className="h-4 w-4" />
  //       <AlertTitle className="font-semibold">Link a Github account</AlertTitle>
  //       <AlertDescription>
  //         You don't have any Github accounts linked. Please{' '}
  //         <a
  //           href="/api/v1/cloud/users/github-app/start"
  //           className="text-indigo-400"
  //         >
  //           link a Github account
  //         </a>{' '}
  //         first.
  //       </AlertDescription>
  //     </Alert>
  //   );
  // }

  // Render the machine type select
  // Keep rendering all machine types but add visual indicators if they're not allowed
  const renderMachineTypeSelectItem = (machine: (typeof machineTypes)[0]) => {
    const computeType: ComputeType = {
      cpuKind: machine.cpuKind,
      cpus: machine.cpus,
      memoryMb: machine.memoryMb,
    };

    const [allowed] = can(managedCompute.selectCompute(computeType));

    return (
      <SelectItem key={machine.title} value={machine.title}>
        {machine.title} {!allowed && '🔒'}
      </SelectItem>
    );
  };

  return (
    <>
      <div className="text-sm text-muted-foreground">
        Define the compute resources for your services.
      </div>
      <Steps className="mt-6">
        <Step title="Build configuration">
          <div className="grid gap-4">
            <div className="text-sm text-muted-foreground">
              Configure the Github repository the service should deploy from.
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
              Configure the runtime settings for this service.
            </div>
            <Label>Environment Variables</Label>
            <EnvGroupArray
              values={secrets}
              setValues={(value) => {
                setSecrets(value);
                setValue(
                  'secrets.add',
                  value.map((item) => ({
                    key: item.key,
                    value: item.value,
                  })),
                );
              }}
            />
            {secretsError && (
              <div className="text-sm text-red-500">{secretsError}</div>
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
                          {machineTypes.map(renderMachineTypeSelectItem)}
                        </SelectContent>
                      </Select>
                    );
                  }}
                />
                {!isComputeAllowed && (
                  <UpgradeMessage
                    feature={`The selected machine type (${machineType})`}
                  />
                )}
                {cpuKindError && (
                  <div className="text-sm text-red-500 dark:text-red-400">
                    {cpuKindError}
                  </div>
                )}
                {cpusError && (
                  <div className="text-sm text-red-500 dark:text-red-400">
                    {cpusError}
                  </div>
                )}
                {memoryMbError && (
                  <div className="text-sm text-red-500 dark:text-red-400">
                    {memoryMbError}
                  </div>
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
                        maxReplicas: Math.min(10, getMaxReplicas),
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
                      <TabsTrigger variant="underlined" value={type} key={type}>
                        {type}
                      </TabsTrigger>
                    ))}
                  </TabsList>
                  <TabsContent value="Static" className="pt-4 grid gap-4">
                    <Label htmlFor="numReplicas">
                      Number of replicas (max: {getMaxReplicas})
                    </Label>
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
                            id="numReplicas"
                            placeholder="1"
                          />
                        );
                      }}
                    />
                    {!isReplicaCountAllowed && (
                      <UpgradeMessage
                        feature={`More than ${getMaxReplicas} replicas`}
                      />
                    )}
                    {numReplicasError && (
                      <div className="text-sm text-red-500 dark:text-red-400">
                        {numReplicasError}
                      </div>
                    )}
                  </TabsContent>
                  <TabsContent value="Autoscaling" className="pt-4 grid gap-4">
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
                              const minValue = parseInt(e.target.value);
                              field.onChange(minValue);

                              setValue(
                                'runtimeConfig.autoscaling.fly.currentReplicas',
                                minValue,
                              );

                              // If min replicas is greater than max replicas, update max replicas
                              const maxReplicas = watch(
                                'runtimeConfig.autoscaling.maxReplicas',
                              );
                              if (maxReplicas < minValue) {
                                setValue(
                                  'runtimeConfig.autoscaling.maxReplicas',
                                  minValue,
                                );
                              }
                            }}
                          />
                        );
                      }}
                    />
                    {autoscalingMinAwakeReplicasError && (
                      <div className="text-sm text-red-500 dark:text-red-400">
                        {autoscalingMinAwakeReplicasError}
                      </div>
                    )}
                    <Label htmlFor="maxReplicas">
                      Max Replicas (max: {getMaxReplicas})
                    </Label>
                    <Controller
                      control={control}
                      name="runtimeConfig.autoscaling.maxReplicas"
                      render={({ field }) => {
                        const minReplicas =
                          watch('runtimeConfig.autoscaling.minAwakeReplicas') ||
                          1;
                        return (
                          <Input
                            {...field}
                            id="maxReplicas"
                            min={minReplicas}
                            type="number"
                            onChange={(e) => {
                              const maxValue = parseInt(e.target.value);
                              // Ensure max replicas is never less than min replicas
                              const validatedMax = Math.max(
                                maxValue,
                                minReplicas,
                              );
                              field.onChange(validatedMax);

                              if (validatedMax !== maxValue) {
                                // If we had to adjust the value, update the input
                                e.target.value = validatedMax.toString();
                              }
                            }}
                          />
                        );
                      }}
                    />
                    {!isAutoscalingMaxReplicasAllowed && (
                      <UpgradeMessage
                        feature={`More than ${getMaxReplicas} max replicas`}
                      />
                    )}
                    {autoscalingMaxReplicasError && (
                      <div className="text-sm text-red-500 dark:text-red-400">
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
                      <div className="text-sm text-red-500 dark:text-red-400">
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
                            <div className="text-sm text-red-500 dark:text-red-400">
                              {autoscalingWaitDurationError}
                            </div>
                          )}
                          <Label htmlFor="rollingWindowDuration">
                            Rolling Window Duration
                          </Label>
                          <div className="text-sm text-muted-foreground">
                            The amount of time to look at utilization metrics
                            for autoscaling. Lower values will lead to faster
                            scale-up and scale-down. Example: 2m (2 minutes), 5m
                            (5 minutes), 1h (1 hour).
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
                            <div className="text-sm text-red-500 dark:text-red-400">
                              {autoscalingRollingWindowDurationError}
                            </div>
                          )}
                          <Label htmlFor="utilizationScaleUpThreshold">
                            Utilization Scale Up Threshold
                          </Label>
                          <div className="text-sm text-muted-foreground">
                            A value between 0 and 1 which represents the
                            utilization threshold at which to scale up. For
                            example, 0.75 means that if the utilization is above
                            75%, scale up.
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
                                    field.onChange(parseFloat(e.target.value));
                                  }}
                                />
                              );
                            }}
                          />
                          {autoscalingUtilizationScaleUpThresholdError && (
                            <div className="text-sm text-red-500 dark:text-red-400">
                              {autoscalingUtilizationScaleUpThresholdError}
                            </div>
                          )}
                          <Label htmlFor="utilizationScaleDownThreshold">
                            Utilization Scale Down Threshold
                          </Label>
                          <div className="text-sm text-muted-foreground">
                            A value between 0 and 1 which represents the
                            utilization threshold at which to scale down. For
                            example, 0.25 means that if the utilization is below
                            25%, scale down.
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
                                    field.onChange(parseFloat(e.target.value));
                                  }}
                                />
                              );
                            }}
                          />
                          {autoscalingUtilizationScaleDownThresholdError && (
                            <div className="text-sm text-red-500 dark:text-red-400">
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
                            <div className="text-sm text-red-500 dark:text-red-400">
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
          </div>
        </Step>
        <Step title="Name">
          <div className="grid gap-4">
            <div className="text-sm text-muted-foreground">
              Give your service a name.
            </div>
            <Label htmlFor="name">Name</Label>
            <Controller
              control={control}
              name="name"
              render={({ field }) => {
                return (
                  <Input
                    {...field}
                    id="name"
                    placeholder="my-awesome-service"
                  />
                );
              }}
            />
            {nameError && (
              <div className="text-sm text-red-500">{nameError}</div>
            )}
          </div>
        </Step>
        <Step title="Review">
          <div className="grid gap-4">
            <div className="text-sm text-muted-foreground">
              Review the settings for this service.
            </div>
            {/* Show invalid configurations and upgrade messages at review stage */}
            {(!isComputeAllowed ||
              !isReplicaCountAllowed ||
              !isAutoscalingMaxReplicasAllowed) && (
              <div className="border border-red-300 dark:border-red-500 bg-red-50 dark:bg-red-900/30 p-4 rounded-md">
                <h3 className="text-red-800 dark:text-red-200 font-medium mb-2">
                  Your configuration requires a plan upgrade
                </h3>
                <ul className="list-disc list-inside text-sm text-red-700 dark:text-red-300 space-y-1">
                  {!isComputeAllowed && (
                    <li>
                      The selected machine type is not available on your current
                      plan
                    </li>
                  )}
                  {!isReplicaCountAllowed && (
                    <li>The number of replicas exceeds your plan's limit</li>
                  )}
                  {!isAutoscalingMaxReplicasAllowed && (
                    <li>
                      The maximum autoscaling replicas exceeds your plan's limit
                    </li>
                  )}
                </ul>
                <div className="mt-3">
                  <a
                    href={getBillingPortalUrl()}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <Button variant="outline" size="sm">
                      <ArrowUpIcon className="h-4 w-4 mr-1" />
                      Upgrade Plan
                    </Button>
                  </a>
                </div>
              </div>
            )}
            <Button
              onClick={handleSubmit(onSubmit)}
              disabled={
                !installation ||
                !repoOwnerName ||
                !branch ||
                !isComputeAllowed ||
                !isReplicaCountAllowed ||
                !isAutoscalingMaxReplicasAllowed
              }
              className="w-fit px-8"
            >
              {isLoading && <PlusIcon className="h-4 w-4 animate-spin" />}
              Create service
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
// URL for the billing portal to upgrade
const getBillingPortalUrl = () => {
  // Replace with your actual billing portal URL or API call
  return '/tenant-settings/billing-and-limits';
};
export const UpgradeMessage = ({ feature }: { feature: string }) => (
  <div className="flex flex-col gap-2 border border-yellow-400 dark:border-yellow-500 bg-yellow-50 dark:bg-yellow-900/30 p-4 rounded-md my-3">
    <div className="flex items-start gap-2">
      <span className="text-yellow-500 dark:text-yellow-400">⚠️</span>
      <p className="text-sm font-medium text-yellow-800 dark:text-yellow-200">
        {feature} is available on higher tier plans. Upgrade to access this
        feature.
      </p>
    </div>
    <a href={getBillingPortalUrl()} target="_blank" rel="noopener noreferrer">
      <Button
        variant="outline"
        size="sm"
        className="text-xs border-yellow-400 dark:border-yellow-500 text-yellow-700 dark:text-yellow-300 hover:bg-yellow-100 dark:hover:bg-yellow-900/50"
      >
        <ArrowUpIcon className="h-3 w-3 mr-1" />
        Upgrade Plan
      </Button>
    </a>
  </div>
);
