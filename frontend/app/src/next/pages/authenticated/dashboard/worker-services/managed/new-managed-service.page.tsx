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
import { RejectReason } from '@/lib/can/shared/permission.base';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { BillingRequired } from './components/billing-required';
import useCan from '@/next/hooks/use-can';
import { managedCompute } from '@/next/lib/can/features/managed-compute.permissions';
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

function ServiceDetailPageContent() {
  const { serviceName = '', workerId } = useParams<{
    serviceName: string;
    workerId?: string;
  }>();
  const navigate = useNavigate();

  const { data: services, create } = useManagedCompute();

  const decodedServiceName = decodeURIComponent(serviceName);

  const { setBreadcrumbs } = useBreadcrumbs();
  const { canWithReason } = useCan();

  const { rejectReason } = canWithReason(managedCompute.create());

  useEffect(() => {
    const breadcrumbs = [
      {
        title: 'Worker Services',
        label: serviceName,
        url: ROUTES.services.new(WorkerType.MANAGED),
      },
    ];

    setBreadcrumbs(breadcrumbs);

    // Clear breadcrumbs when this component unmounts
    return () => {
      setBreadcrumbs([]);
    };
  }, [decodedServiceName, setBreadcrumbs, serviceName]);

  // Only show BillingRequired if there are no managed workers AND billing is required
  const hasExistingWorkers = (services?.length || 0) > 0;

  const [secrets, setSecrets] = useState<UpdateManagedWorkerSecretRequest>({
    add: [],
    update: [],
    delete: [],
  });

  const [githubRepo, setGithubRepo] = useState<GithubRepoSelectorValue>({
    githubInstallationId: '',
    githubRepositoryOwner: '',
    githubRepositoryName: '',
    githubRepositoryBranch: '',
  });

  const [buildConfig, setBuildConfig] = useState<BuildConfigValue>({
    buildDir: './',
    dockerfilePath: './Dockerfile',
    serviceName: '',
  });

  const [machineConfig, setMachineConfig] = useState<MachineConfigValue>({
    cpuKind: 'shared',
    cpus: 1,
    memoryMb: 1024,
    regions: [ManagedWorkerRegion.Ewr],
  });

  const [isDeploying, setIsDeploying] = useState(false);

  const handleDeploy = async () => {
    if (!githubRepo.githubInstallationId || !githubRepo.githubRepositoryName) {
      return;
    }

    setIsDeploying(true);
    try {
      await create.mutateAsync({
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
      console.error('Failed to deploy service:', error);
    } finally {
      setIsDeploying(false);
    }
  };

  if (rejectReason == RejectReason.BILLING_REQUIRED && !hasExistingWorkers) {
    return <BillingRequired />;
  }

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Manage workers in a worker service">
          New Managed Worker Service
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
            type="create"
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
            type="create"
          />
        </div>
      </div>
    </BasicLayout>
  );
}

export default function ServiceDetailPage() {
  return (
    <ManagedComputeProvider>
      <WorkersProvider>
        <ServiceDetailPageContent />
      </WorkersProvider>
    </ManagedComputeProvider>
  );
}
