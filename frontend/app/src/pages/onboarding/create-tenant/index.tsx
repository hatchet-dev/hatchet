import { HeroPanel } from '../../auth/components/hero-panel';
import useCloud from '../../auth/hooks/use-cloud';
import { TenantCreateForm } from './components/tenant-create-form';
import { OnboardingFormData } from './types';
import { Button } from '@/components/v1/ui/button';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { useAnalytics } from '@/hooks/use-analytics';
import { useOrganizations } from '@/hooks/use-organizations';
import { usePendingInvites } from '@/hooks/use-pending-invites';
import api, {
  CreateTenantRequest,
  queries,
  Tenant,
  TenantEnvironment,
} from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { OrganizationTenant } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { useSearchParams } from '@/lib/router-helpers';
import { AppContextProvider } from '@/providers/app-context';
import { appRoutes } from '@/router';
import {
  QuestionMarkCircleIcon,
  ChevronDownIcon,
  ChevronUpIcon,
} from '@heroicons/react/24/outline';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { useMemo, useState, useEffect } from 'react';

function CreateTenantInner() {
  const [searchParams] = useSearchParams();
  const { organizationData, isCloudEnabled } = useOrganizations();
  const { cloud } = useCloud();
  const [showHelp, setShowHelp] = useState(false);
  const { capture } = useAnalytics();
  const { pendingInvitesQuery, isLoading: isPendingInvitesLoading } =
    usePendingInvites();

  const organizationId = searchParams.get('organizationId');

  // Track page view
  useEffect(() => {
    capture('onboarding_create_tenant_viewed');
  }, [capture]);

  const [selectedOrganizationId, setSelectedOrganizationId] = useState<
    string | null
  >(null);

  // Auto-select organization logic
  useEffect(() => {
    if (organizationData?.rows) {
      const availableOrgs = organizationData.rows.filter((org) => org.isOwner);

      // If organizationId from URL is valid, use it
      if (
        organizationId &&
        availableOrgs.find((org) => org.metadata.id === organizationId)
      ) {
        setSelectedOrganizationId(organizationId);
      }
      // If there's only one organization, auto-select it
      else if (availableOrgs.length === 1) {
        setSelectedOrganizationId(availableOrgs[0].metadata.id);
      }
      // Otherwise, leave it null to show placeholder (ignoring invalid org IDs)
    }
  }, [organizationData, organizationId]);

  const [formData, setFormData] = useState<OnboardingFormData>({
    name: '',
    slug: '',
    environment: TenantEnvironment.Development,
    tenantData: {
      name: '',
      environment: TenantEnvironment.Development,
      referralSource: '',
    },
    referralSource: '',
  });
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });
  const navigate = useNavigate();

  useEffect(() => {
    if (
      !isPendingInvitesLoading &&
      pendingInvitesQuery.data &&
      pendingInvitesQuery.data > 0
    ) {
      navigate({ to: appRoutes.onboardingInvitesRoute.to, replace: true });
    }
  }, [isPendingInvitesLoading, pendingInvitesQuery.data, navigate]);

  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  const existingTenantNames = useMemo(() => {
    return (listMembershipsQuery.data?.rows ?? [])
      .map((m) => m.tenant?.name)
      .filter((n): n is string => Boolean(n && n.trim().length > 0));
  }, [listMembershipsQuery.data?.rows]);

  const createMutation = useMutation({
    mutationKey: ['user:update:login'],
    mutationFn: async (data: CreateTenantRequest) => {
      // Use cloud API if cloud is enabled and organization is selected
      if (cloud && selectedOrganizationId) {
        const result = await cloudApi.organizationCreateTenant(
          selectedOrganizationId,
          {
            name: data.name,
            slug: data.slug,
          },
        );

        return { type: 'cloud', data: result.data };
      } else {
        // Use regular API for self-hosted
        const tenant = await api.tenantCreate(data);

        return { type: 'regular', data: tenant.data };
      }
    },
    onSuccess: async (result) => {
      await listMembershipsQuery.refetch();

      // Track tenant creation
      capture('onboarding_tenant_created', {
        tenant_type: result.type,
        is_cloud: result.type === 'cloud',
      });

      setTimeout(() => {
        if (result.type === 'cloud') {
          const tenant = result.data as OrganizationTenant;
          navigate({
            to: appRoutes.tenantOverviewRoute.to,
            params: { tenant: tenant.id },
          });
          return;
        }

        const tenant = result.data as Tenant;
        navigate({
          to: appRoutes.tenantOverviewRoute.to,
          params: { tenant: tenant.metadata.id },
        });
      }, 0);
    },
    onError: handleApiError,
  });

  const validateForm = (): boolean => {
    const { name } = formData.tenantData;
    const errors: Record<string, string> = {};

    // Basic validation for name
    if (!name || name.length < 1 || name.length > 32) {
      errors.name = 'Name must be between 1 and 32 characters';
    }

    if (isCloudEnabled && !selectedOrganizationId) {
      errors.organizationId = 'Please select an organization';
    }

    // Set errors if any exist
    setFieldErrors(errors);

    // Return false if there are any errors
    return Object.keys(errors).length === 0;
  };

  const generateSlug = (name: string): string => {
    return (
      name
        .toLowerCase()
        .replace(/[^a-z0-9-]/g, '-')
        .replace(/-+/g, '-')
        .replace(/^-|-$/g, '') +
      '-' +
      Math.random().toString(36).substring(0, 5)
    );
  };

  const handleTenantCreate = (tenantData: {
    name: string;
    environment: TenantEnvironment;
    referralSource?: string;
  }) => {
    if (!validateForm()) {
      return;
    }

    // Generate slug from name
    const slug = generateSlug(tenantData.name);

    // Build onboarding data object
    const onboardingData: Record<string, any> = {};
    if (tenantData.referralSource && tenantData.referralSource.trim() !== '') {
      onboardingData.referral_source = tenantData.referralSource;
    }

    createMutation.mutate({
      name: tenantData.name,
      slug,
      environment: tenantData.environment,
      onboardingData:
        Object.keys(onboardingData).length > 0 ? onboardingData : undefined,
    });
  };

  const updateFormData = (value: {
    name: string;
    environment: string;
    referralSource?: string;
  }) => {
    setFormData({
      ...formData,
      tenantData: {
        name: value.name,
        environment: value.environment as TenantEnvironment,
        referralSource: value.referralSource,
      },
      name: value.name,
      environment: value.environment as TenantEnvironment,
      referralSource: value.referralSource,
    });
  };

  return (
    <div className="min-h-screen w-full lg:grid lg:grid-cols-2">
      <div className="relative hidden overflow-hidden bg-muted/30 px-10 py-12 lg:flex">
        <div className="pointer-events-none absolute inset-0 bg-gradient-to-br from-primary/10 via-transparent to-transparent" />
        <HeroPanel />
      </div>

      <div className="w-full overflow-y-auto">
        <div className="flex min-h-screen w-full items-start justify-center px-4 py-10 lg:justify-start lg:px-12 lg:py-12">
          <div className="w-full max-w-lg space-y-6">
            <div className="flex flex-col space-y-2 text-center">
              <div className="flex items-center justify-center gap-2">
                <h1 className="text-2xl font-semibold tracking-tight">
                  Create a new tenant
                </h1>
                <TooltipProvider delayDuration={200}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button type="button" className="inline-flex">
                        <QuestionMarkCircleIcon className="h-5 w-5 text-muted-foreground cursor-help" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>
                        A tenant is an isolated environment for your data and
                        workflows.
                      </p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
              <div className="space-y-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowHelp(!showHelp)}
                  className="text-xs text-muted-foreground hover:text-foreground"
                >
                  Trying to join an existing tenant?
                  {showHelp ? (
                    <ChevronUpIcon className="ml-1 h-3 w-3" />
                  ) : (
                    <ChevronDownIcon className="ml-1 h-3 w-3" />
                  )}
                </Button>
                {showHelp && (
                  <div className="rounded-lg border bg-muted/50 p-4 text-left text-sm text-muted-foreground">
                    <p className="mb-2">
                      If you're trying to join an existing tenant, you should
                      not create a new one. Some reasons that you may
                      accidentally end up here are:
                    </p>
                    <ul className="list-disc space-y-1 pl-5">
                      <li>You're logging in with the wrong email address</li>
                      <li>Your Hatchet account is at a different URL</li>
                      <li>You need an invitation from a colleague</li>
                    </ul>
                  </div>
                )}
              </div>
            </div>

            <TenantCreateForm
              value={formData.tenantData}
              onChange={updateFormData}
              onNext={() => handleTenantCreate(formData.tenantData)}
              isLoading={createMutation.isPending}
              fieldErrors={fieldErrors}
              formData={formData}
              setFormData={setFormData}
              className=""
              organizationList={organizationData}
              selectedOrganizationId={selectedOrganizationId}
              onOrganizationChange={setSelectedOrganizationId}
              isCloudEnabled={isCloudEnabled}
              existingTenantNames={existingTenantNames}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

export default function CreateTenant() {
  return (
    <AppContextProvider>
      <CreateTenantInner />
    </AppContextProvider>
  );
}
