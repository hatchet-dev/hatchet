import api, {
  CreateTenantRequest,
  queries,
  TenantEnvironment,
  TenantVersion,
} from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { TenantCreateForm } from './components/tenant-create-form';
import { useTenant } from '@/lib/atoms';
import { Button } from '@/components/ui/button';
import { HearAboutUsForm } from './components/hear-about-us-form';
import { WhatBuildingForm } from './components/what-building-form';
import { StepProgress } from './components/step-progress';
import { OnboardingStepConfig, OnboardingFormData } from './types';
import { useAnalytics } from '@/hooks/use-analytics';

export default function CreateTenant() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const cloudMeta = useCloudApiMeta();

  const stepFromUrl = parseInt(searchParams.get('step') || '0', 10);

  const [currentStep, setCurrentStep] = useState(
    Math.max(0, Math.min(stepFromUrl, 2)),
  );
  const [selectedOrganizationId, setSelectedOrganizationId] = useState<string>('');

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
  const { setTenant } = useTenant();
  const { capture } = useAnalytics();

  // Get organization list when cloud is enabled
  const organizationListQuery = useQuery({
    queryKey: ['organization:list'],
    queryFn: async () => {
      const result = await cloudApi.organizationList();
      return result.data;
    },
    enabled: !!cloudMeta?.data,
  });

  // Sync currentStep with URL parameter
  useEffect(() => {
    const stepFromUrl = parseInt(searchParams.get('step') || '0', 10);
    const validStep = Math.max(0, Math.min(stepFromUrl, 2));
    setCurrentStep(validStep);
  }, [searchParams]);

  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  const createMutation = useMutation({
    mutationKey: ['user:update:login'],
    mutationFn: async (data: CreateTenantRequest) => {
      // Use cloud API if cloud is enabled and organization is selected
      if (cloudMeta?.data && selectedOrganizationId) {
        const result = await cloudApi.organizationCreateTenant(selectedOrganizationId, {
          name: data.name,
          slug: data.slug,
        });

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
          // For cloud tenants, redirect back to organization page
          window.location.href = `/organizations/${selectedOrganizationId}`;
        } else {
          // For regular tenants, use the existing onboarding flow
          const tenant = result.data as any;
          if (tenant.version === TenantVersion.V1) {
            window.location.href = `/tenants/${tenant.metadata.id}/onboarding/get-started`;
          } else {
            window.location.href = `/onboarding/get-started?tenant=${tenant.metadata.id}`;
          }
        }
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
      navigate(`?step=${nextStep}`, { replace: false });
    }
  };

  // Pure validation function for button state (no side effects)
  const isCurrentStepValid = (): boolean => {
    if (currentStep === 2) {
      const { name } = formData.tenantData;

      // Basic validation for name
      if (!name || name.length < 4 || name.length > 32) {
        return false;
      }

      // Validate organization selection for cloud mode
      if (cloudMeta?.data && !selectedOrganizationId) {
        return false;
      }
    }

    return true;
  };

  const validateCurrentStep = (): boolean => {
    // For the tenant create form (step 2), we need to validate the form
    if (currentStep === 2) {
      const { name } = formData.tenantData;
      const errors: Record<string, string> = {};

      // Basic validation for name
      if (!name || name.length < 4 || name.length > 32) {
        errors.name = 'Name must be between 4 and 32 characters';
      }

      // Validate organization selection for cloud mode
      if (cloudMeta?.data && !selectedOrganizationId) {
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
      navigate(`?step=${previousStep}`, { replace: false });
    }
  };

  const handleStepClick = (stepIndex: number) => {
    // Allow navigation to any step within valid range
    if (stepIndex >= 0 && stepIndex < steps.length) {
      navigate(`?step=${stepIndex}`, { replace: false });
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
    // Validate organization selection for cloud mode
    if (cloudMeta?.data && !selectedOrganizationId) {
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
            currentStep === 2
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
          {...(currentStep === 2 && {
            organizationList: organizationListQuery.data,
            selectedOrganizationId: selectedOrganizationId,
            onOrganizationChange: setSelectedOrganizationId,
            isCloudEnabled: !!cloudMeta?.data,
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
                navigate(`?step=${currentStep + 1}`, { replace: false })
              }
            >
              Skip
            </Button>
          ) : (
            <Button
              variant="default"
              onClick={() => {
                if (currentStep === 2) {
                  // For tenant create form, validate and submit
                  if (validateCurrentStep()) {
                    handleTenantCreate(formData.tenantData);
                  }
                } else {
                  handleNext();
                }
              }}
              disabled={createMutation.isPending || (currentStep === 2 && !isCurrentStepValid())}
            >
              {createMutation.isPending ? (
                <div className="flex items-center gap-2">
                  <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
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
    <div className="flex flex-col items-center justify-center min-h-full w-full p-4">
      <div className="w-full max-w-[450px] space-y-6">
        {renderCurrentStep()}
      </div>
    </div>
  );
}
