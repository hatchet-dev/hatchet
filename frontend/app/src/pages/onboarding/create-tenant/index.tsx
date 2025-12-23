import useCloud from '../../auth/hooks/use-cloud';
import { HearAboutUsForm } from './components/hear-about-us-form';
import { StepProgress } from './components/step-progress';
import { TenantCreateForm } from './components/tenant-create-form';
import { WhatBuildingForm } from './components/what-building-form';
import { OnboardingStepConfig, OnboardingFormData } from './types';
import { Button } from '@/components/v1/ui/button';
import { useAnalytics } from '@/hooks/use-analytics';
import { useOrganizations } from '@/hooks/use-organizations';
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
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { useState, useEffect } from 'react';

const FINAL_STEP = 2;

export default function CreateTenant() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { organizationData, isCloudEnabled } = useOrganizations();
  const { cloud } = useCloud();

  const getValidatedStep = (stepParam: string | null): number => {
    if (stepParam === null) {
      return 0;
    }

    // Handle numbers that may be wrapped in quotes (e.g. "%221%22")
    const normalized =
      typeof stepParam === 'string'
        ? stepParam.replace(/["']/g, '')
        : stepParam;

    const parsedStep = Number.parseInt(normalized, 10);
    if (!Number.isFinite(parsedStep)) {
      return 0;
    }
    return Math.max(0, Math.min(parsedStep, FINAL_STEP));
  };

  const stepFromUrl = getValidatedStep(searchParams.get('step'));
  const organizationId = searchParams.get('organizationId');

  const [currentStep, setCurrentStep] = useState(stepFromUrl);
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
    hearAboutUs: [],
    whatBuilding: [],
    environment: TenantEnvironment.Development,
    tenantData: { name: '', environment: TenantEnvironment.Development },
  });
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });
  const { capture } = useAnalytics();
  const navigate = useNavigate();
  // Sync currentStep with URL parameter
  useEffect(() => {
    const stepFromUrl = getValidatedStep(searchParams.get('step'));
    setCurrentStep(stepFromUrl);
  }, [searchParams]);

  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

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

        // Track onboarding analytics for cloud tenant
        capture('onboarding_completed', {
          hear_about_us: Array.isArray(formData.hearAboutUs)
            ? formData.hearAboutUs.join(', ')
            : formData.hearAboutUs,
          what_building: Array.isArray(formData.whatBuilding)
            ? formData.whatBuilding.join(', ')
            : formData.whatBuilding,
          tenant_id: result.data.id,
          organization_id: selectedOrganizationId,
        });

        return { type: 'cloud', data: result.data };
      } else {
        // Use regular API for self-hosted
        const tenant = await api.tenantCreate(data);

        // Track onboarding analytics for regular tenant
        capture('onboarding_completed', {
          hear_about_us: Array.isArray(formData.hearAboutUs)
            ? formData.hearAboutUs.join(', ')
            : formData.hearAboutUs,
          what_building: Array.isArray(formData.whatBuilding)
            ? formData.whatBuilding.join(', ')
            : formData.whatBuilding,
          tenant_id: tenant.data.metadata.id,
          organization_id: null,
        });

        return { type: 'regular', data: tenant.data };
      }
    },
    onSuccess: async (result) => {
      await listMembershipsQuery.refetch();

      setTimeout(() => {
        if (result.type === 'cloud') {
          const tenant = result.data as OrganizationTenant;
          navigate({
            to: appRoutes.tenantOnboardingGetStartedRoute.to,
            params: { tenant: tenant.id },
          });
          return;
        }

        const tenant = result.data as Tenant;
        navigate({
          to: appRoutes.tenantOnboardingGetStartedRoute.to,
          params: { tenant: tenant.metadata.id },
        });
      }, 0);
    },
    onError: handleApiError,
  });

  const steps: OnboardingStepConfig[] = [
    {
      title: 'What are you building?',
      subtitle: 'Help us personalize your onboarding experience',
      component: WhatBuildingForm,
      canSkip: true,
      key: 'whatBuilding',
    },
    {
      title: 'Where did you hear about Hatchet?',
      component: HearAboutUsForm,
      canSkip: true,
      key: 'hearAboutUs',
    },
    {
      title: 'Create a new tenant',
      subtitle:
        'A tenant is an isolated environment for your data and workflows.',
      component: TenantCreateForm,
      canSkip: false,
      key: 'tenantData',
      buttonLabel: 'Create Tenant',
    },
  ];

  const handleNext = () => {
    if (currentStep < steps.length - 1) {
      const nextStep = currentStep + 1;
      setCurrentStep(nextStep);
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set('step', String(nextStep));
        return next;
      });
    }
  };

  // Pure validation function for button state (no side effects)
  const isCurrentStepValid = (): boolean => {
    if (currentStep === FINAL_STEP) {
      const { name } = formData.tenantData;

      // Basic validation for name
      if (!name || name.length < 4 || name.length > 32) {
        return false;
      }

      if (isCloudEnabled && !selectedOrganizationId) {
        return false;
      }
    }

    return true;
  };

  const validateCurrentStep = (): boolean => {
    // For the tenant create form (step 2), we need to validate the form
    if (currentStep === FINAL_STEP) {
      const { name } = formData.tenantData;
      const errors: Record<string, string> = {};

      // Basic validation for name
      if (!name || name.length < 4 || name.length > 32) {
        errors.name = 'Name must be between 4 and 32 characters';
      }

      if (isCloudEnabled && !selectedOrganizationId) {
        errors.organizationId = 'Please select an organization';
      }

      // Set errors if any exist
      setFieldErrors(errors);

      // Return false if there are any errors
      return Object.keys(errors).length === 0;
    }

    return true;
  };

  const handlePrevious = () => {
    if (currentStep > 0) {
      const previousStep = currentStep - 1;
      setCurrentStep(previousStep);
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set('step', String(previousStep));
        return next;
      });
    }
  };

  const handleStepClick = (stepIndex: number) => {
    // Allow navigation to any step within valid range
    if (stepIndex >= 0 && stepIndex < steps.length) {
      setCurrentStep(stepIndex);
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set('step', String(stepIndex));
        return next;
      });
    }
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
  }) => {
    if (isCloudEnabled && !selectedOrganizationId) {
      setFieldErrors({
        organizationId: 'Please select an organization',
      });
      return;
    }

    // Clear any previous errors
    setFieldErrors({});

    // Generate slug from name
    const slug = generateSlug(tenantData.name);

    // Prepare the onboarding data to send with the tenant creation request
    const onboardingData = {
      hearAboutUs: Array.isArray(formData.hearAboutUs)
        ? formData.hearAboutUs.join(', ')
        : formData.hearAboutUs,
      whatBuilding: Array.isArray(formData.whatBuilding)
        ? formData.whatBuilding.join(', ')
        : formData.whatBuilding,
    };

    createMutation.mutate({
      name: tenantData.name,
      slug,
      onboardingData,
      environment: tenantData.environment,
    });
  };

  const updateFormData = (key: keyof OnboardingFormData, value: any) => {
    setFormData({ ...formData, [key]: value });

    // Sync tenantData with name for backward compatibility
    if (key === 'tenantData') {
      setFormData({
        ...formData,
        [key]: value,
        name: value.name,
      });
    }
  };

  const renderCurrentStep = () => {
    const currentStepConfig = steps[currentStep];
    const StepComponent = currentStepConfig.component;

    return (
      <>
        <div className="flex flex-col space-y-2 text-center">
          <h1 className="text-2xl font-semibold tracking-tight">
            {currentStepConfig.title}
          </h1>
          {currentStepConfig.subtitle && (
            <p className="text-sm text-muted-foreground">
              {currentStepConfig.subtitle}
            </p>
          )}
        </div>

        <StepComponent
          value={formData[currentStepConfig.key]}
          onChange={(value) => updateFormData(currentStepConfig.key, value)}
          onNext={
            currentStep === FINAL_STEP
              ? () => handleTenantCreate(formData.tenantData)
              : handleNext
          }
          onPrevious={handlePrevious}
          isLoading={createMutation.isPending}
          fieldErrors={fieldErrors}
          formData={formData}
          setFormData={setFormData}
          className=""
          // Organization-related props only for TenantCreateForm (step 2)
          {...(currentStep === FINAL_STEP && {
            organizationList: organizationData,
            selectedOrganizationId: selectedOrganizationId,
            onOrganizationChange: setSelectedOrganizationId,
            isCloudEnabled,
          })}
        />

        <div className="flex justify-between">
          <StepProgress
            steps={steps}
            currentStep={currentStep}
            onStepClick={handleStepClick}
          />

          {currentStepConfig.canSkip ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                setSearchParams((prev) => {
                  const next = new URLSearchParams(prev);
                  next.set('step', String(currentStep + 1));
                  return next;
                })
              }
            >
              Skip
            </Button>
          ) : (
            <Button
              variant="default"
              onClick={() => {
                if (currentStep === FINAL_STEP) {
                  // For tenant create form, validate and submit
                  if (validateCurrentStep()) {
                    handleTenantCreate(formData.tenantData);
                  }
                } else {
                  handleNext();
                }
              }}
              disabled={
                createMutation.isPending ||
                (currentStep === FINAL_STEP && !isCurrentStepValid())
              }
            >
              {createMutation.isPending ? (
                <div className="flex items-center gap-2">
                  <div className="size-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  Creating...
                </div>
              ) : (
                currentStepConfig.buttonLabel || 'Next'
              )}
            </Button>
          )}
        </div>
      </>
    );
  };

  return (
    <div className="flex min-h-full w-full flex-col items-center justify-center p-4">
      <div className="w-full max-w-[450px] space-y-6">
        {renderCurrentStep()}
      </div>
    </div>
  );
}
