import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
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
import { EnvVars } from './components/config/env-vars';
import { SecretsEditor } from './components/config/build-config';
import { RuntimeConfig } from './components/config/runtime-config';
import { UpdateManagedWorkerSecretRequest } from '@/lib/api/generated/cloud/data-contracts';

function ServiceDetailPageContent() {
  const { serviceName = '', workerId } = useParams<{
    serviceName: string;
    workerId?: string;
  }>();

  const { data: services } = useManagedCompute();

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
            <DocsButton doc={docs.home.workers} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
      <EnvVars />
      <SecretsEditor
        secrets={secrets}
        setSecrets={setSecrets}
        original={{
          directSecrets: [
            {
              key: 'SECRET_KEY',
              hint: 'This is a secret *****',
              id: '123',
            },
            {
              key: 'SECRET_KEY_2',
              hint: 'This is a secret *****',
              id: '456',
            },
          ],
          globalSecrets: [
            {
              key: 'GLOBAL_SECRET',
              hint: 'This is a global secret *****',
              id: '789',
            },
          ],
        }}
      />
      <pre>{JSON.stringify(secrets, null, 2)}</pre>
      <RuntimeConfig />
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
