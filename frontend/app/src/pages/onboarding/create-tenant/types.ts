import { TenantEnvironment } from '@/lib/api';

// Base interface that all onboarding step components must implement
export interface OnboardingStepProps<T> {
  value: T;
  onChange: (value: T) => void;
  onNext?: () => void;
  onPrevious?: () => void;
  isLoading?: boolean;
  fieldErrors?: Record<string, string>;
  formData?: OnboardingFormData;
  setFormData?: (data: OnboardingFormData) => void;
  className?: string;
}

// Form data interface for the entire onboarding flow
export interface OnboardingFormData {
  name: string;
  slug: string;
  hearAboutUs: string | string[];
  whatBuilding: string | string[];
  environment: TenantEnvironment;
  tenantData: { name: string; environment: TenantEnvironment };
}

// Step configuration interface
export interface OnboardingStepConfig<T, C> {
  title: string;
  subtitle?: string;
  component:
    | React.ComponentType<OnboardingStepProps<T>>
    | React.ComponentType<C>;
  canSkip: boolean;
  key: keyof OnboardingFormData;
  buttonLabel?: string;
}
