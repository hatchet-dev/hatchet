import api, {
  CreateTenantRequest,
  queries,
  TenantEnvironment,
  TenantVersion,
} from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { TenantCreateForm } from './components/tenant-create-form';
import { useTenant } from '@/lib/atoms';
import { Button } from '@/components/ui/button';
import { HearAboutUsForm } from './components/hear-about-us-form';
import { WhatBuildingForm } from './components/what-building-form';
import { StepProgress } from './components/step-progress';
import { OnboardingStepConfig, OnboardingFormData } from './types';

export default function CreateTenant() {
  const [currentStep, setCurrentStep] = useState(0);
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

  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  const createMutation = useMutation({
    mutationKey: ['user:update:login'],
    mutationFn: async (data: CreateTenantRequest) => {
      const tenant = await api.tenantCreate(data);

      // Track onboarding analytics
      if (typeof window !== 'undefined' && (window as any).posthog) {
        (window as any).posthog.capture('onboarding_completed', {
          hear_about_us: Array.isArray(formData.hearAboutUs)
            ? formData.hearAboutUs.join(', ')
            : formData.hearAboutUs,
          what_building: Array.isArray(formData.whatBuilding)
            ? formData.whatBuilding.join(', ')
            : formData.whatBuilding,
          tenant_id: tenant.data.metadata.id,
        });
      }

      return tenant.data;
    },
    onSuccess: async (tenant) => {
      setTenant(tenant);
      await listMembershipsQuery.refetch();

      // Hack to wait for next event loop tick so local storage is updated
      setTimeout(() => {
        if (tenant.version === TenantVersion.V1) {
          window.location.href = `/tenants/${tenant.metadata.id}/onboarding/get-started`;
        } else {
          window.location.href = `/onboarding/get-started?tenant=${tenant.metadata.id}`;
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
      setCurrentStep(currentStep + 1);
    }
  };

  const validateCurrentStep = (): boolean => {
    // For the tenant create form (step 2), we need to validate the form
    if (currentStep === 2) {
      const { name } = formData.tenantData;

      // Clear previous errors
      setFieldErrors({});

      // Basic validation
      if (!name || name.length < 4 || name.length > 32) {
        setFieldErrors((prev) => ({
          ...prev,
          name: 'Name must be between 4 and 32 characters',
        }));
        return false;
      }

      return true;
    }

    return true;
  };

  const handlePrevious = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1);
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
        />

        <div className="flex justify-between">
          <StepProgress steps={steps} currentStep={currentStep} />

          {currentStepConfig.canSkip ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentStep(2)}
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
              disabled={createMutation.isPending}
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
    <div className="flex flex-row flex-1 w-full h-full">
      <div className="container relative flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
        <div className="lg:p-8 mx-auto w-screen">
          <div className="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[450px]">
            {renderCurrentStep()}
          </div>
        </div>
      </div>
    </div>
  );
}
