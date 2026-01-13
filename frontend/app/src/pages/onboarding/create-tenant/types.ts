import { TenantEnvironment } from '@/lib/api';

// Base interface that all onboarding step components must implement
export interface OnboardingStepProps<T = any> {
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
  environment: TenantEnvironment;
  tenantData: {
    name: string;
    environment: TenantEnvironment;
    referralSource?: string;
  };
  referralSource?: string;
}

// Step configuration interface
export interface OnboardingStepConfig {
  title: string;
  subtitle?: string;
  component:
    | React.ComponentType<OnboardingStepProps>
    | React.ComponentType<any>;
  canSkip: boolean;
  key: keyof OnboardingFormData;
  validate?: (value: any) => boolean;
  buttonLabel?: string;
}
