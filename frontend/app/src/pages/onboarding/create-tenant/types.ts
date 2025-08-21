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
  hearAboutUs: string;
  whatBuilding: string;
  tenantData: { name: string; slug: string };
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

// Base component type that all steps should extend
export type OnboardingStepComponent<T = any> = React.ComponentType<
  OnboardingStepProps<T>
>;
