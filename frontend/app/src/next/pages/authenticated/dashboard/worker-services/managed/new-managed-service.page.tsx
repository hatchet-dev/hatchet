import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { WorkersProvider } from '@/next/hooks/use-workers';
import { useBreadcrumbs } from '@/next/hooks/use-breadcrumbs';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  PageTitle,
  HeadlineActions,
  HeadlineActionItem,
} from '@/next/components/ui/page-header';
import docs from '@/next/lib/docs';
import { ROUTES } from '@/next/lib/routes';
import { WorkerType } from '@/lib/api';
import {
  useManagedCompute,
  ManagedComputeProvider,
} from '@/next/hooks/use-managed-compute';
import { RejectReason } from '@/next/lib/can/shared/permission.base';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { CloudOnly } from './components/cloud-only';
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
import { Step, Steps } from '@/components/v1/ui/steps';
import { Button } from '@/next/components/ui/button';
import { BillingRequired } from './components/billing-required';
import { useTenant } from '@/next/hooks/use-tenant';
function ServiceDetailPageContent() {
  const navigate = useNavigate();

  const { tenant } = useTenant();

  const { data: services, create } = useManagedCompute();

  const breadcrumb = useBreadcrumbs();

  useEffect(() => {
    breadcrumb.set([
      {
        title: 'Worker Services',
        label: 'New Managed Worker Service',
        url: ROUTES.services.new(tenant?.metadata.id || '', WorkerType.MANAGED),
      },
    ]);
  }, [breadcrumb]);

  const { canWithReason } = useCan();

  const { rejectReason } = canWithReason(managedCompute.create());

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
    numReplicas: 1,
  });

  const [isDeploying, setIsDeploying] = useState(false);

  const [activeStep, setActiveStep] = useState(0);

  const handleDeploy = async () => {
    if (!githubRepo.githubInstallationId || !githubRepo.githubRepositoryName) {
      return;
    }

    setIsDeploying(true);
    try {
      const deployedService = await create.mutateAsync({
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
        ROUTES.services.detail(
          tenant?.metadata.id || '',
          deployedService.metadata.id,
          WorkerType.MANAGED,
        ),
      );
    } catch (error) {
      console.error('Failed to deploy service:', error);
    } finally {
      setIsDeploying(false);
    }
  };

  const handleNext = () => {
    setActiveStep((prev) => Math.min(prev + 1, 4));
  };

  const handlePrevious = () => {
    setActiveStep((prev) => Math.max(prev - 1, 0));
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
      <Separator className="mt-4" />
      <div className="flex flex-col gap-4">
        <Steps>
          <Step
            title="GitHub Repository"
            open={activeStep === 0}
            setOpen={(open: boolean) => setActiveStep(open ? 0 : -1)}
          >
            <GithubIntegrationProvider>
              <GithubRepoSelector
                value={githubRepo}
                onChange={(value) => {
                  setGithubRepo(value);
                }}
              />
            </GithubIntegrationProvider>
            <div className="flex justify-end mt-4">
              <Button onClick={handleNext}>Next</Button>
            </div>
          </Step>
          <Step
            title="Build Configuration"
            open={activeStep === 1}
            setOpen={(open: boolean) => setActiveStep(open ? 1 : -1)}
          >
            <BuildConfig
              githubRepo={githubRepo}
              value={buildConfig}
              onChange={(value) => {
                setBuildConfig(value);
              }}
              type="create"
            />
            <div className="flex justify-between mt-4">
              <Button variant="outline" onClick={handlePrevious}>
                Previous
              </Button>
              <Button onClick={handleNext}>Next</Button>
            </div>
          </Step>
          <Step
            title="Environment Variables"
            open={activeStep === 2}
            setOpen={(open: boolean) => setActiveStep(open ? 2 : -1)}
          >
            <EnvVarsEditor
              secrets={secrets}
              setSecrets={setSecrets}
              original={{
                directSecrets: [],
                globalSecrets: [],
              }}
            />
            <div className="flex justify-between mt-4">
              <Button variant="outline" onClick={handlePrevious}>
                Previous
              </Button>
              <Button onClick={handleNext}>Next</Button>
            </div>
          </Step>
          <Step
            title="Machine Configuration"
            open={activeStep === 3}
            setOpen={(open: boolean) => setActiveStep(open ? 3 : -1)}
          >
            <MachineConfig
              config={machineConfig}
              setConfig={(value) => {
                setMachineConfig(value);
              }}
            />
            <div className="flex justify-between mt-4">
              <Button variant="outline" onClick={handlePrevious}>
                Previous
              </Button>
              <Button onClick={handleNext}>Next</Button>
            </div>
          </Step>
          <Step
            title="Review & Deploy"
            open={activeStep === 4}
            setOpen={(open: boolean) => setActiveStep(open ? 4 : -1)}
          >
            <Summary
              githubRepo={githubRepo}
              buildConfig={buildConfig}
              machineConfig={machineConfig}
              secrets={secrets}
              type="create"
            />
            <div className="flex justify-between mt-4">
              <Button variant="outline" onClick={handlePrevious}>
                Previous
              </Button>
              <Button onClick={handleDeploy} disabled={isDeploying}>
                {isDeploying ? 'Deploying...' : 'Deploy Service'}
              </Button>
            </div>
          </Step>
        </Steps>
      </div>
    </BasicLayout>
  );
}

export default function ServiceDetailPage() {
  const { canWithReason } = useCan();

  const { rejectReason } = canWithReason(managedCompute.create());

  if (rejectReason == RejectReason.CLOUD_ONLY) {
    return <CloudOnly />;
  }

  return (
    <ManagedComputeProvider>
      <WorkersProvider>
        <ServiceDetailPageContent />
      </WorkersProvider>
    </ManagedComputeProvider>
  );
}
