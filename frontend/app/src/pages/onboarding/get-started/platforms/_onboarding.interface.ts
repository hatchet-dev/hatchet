export interface OnboardingInterface {
  setup: React.FC<{ existingProject: boolean }>;
  worker: React.FC;
}
