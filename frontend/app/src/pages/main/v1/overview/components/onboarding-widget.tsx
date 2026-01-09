import { RadialProgressBar } from './radial-progress-bar';
import { cn } from '@/lib/utils';

interface OnboardingWidgetProps {
  steps: number;
  currentStep: number;
  label: string;
  className?: string;
}

export const OnboardingWidget = ({
  steps,
  currentStep,
  label,
  className,
}: OnboardingWidgetProps) => {
  return (
    <div
      className={cn(
        'inline-flex items-center gap-2 rounded-md border bg-muted/50 px-3 py-1.5 text-sm',
        className,
      )}
    >
      <span className="font-medium text-muted-foreground">{label}</span>
      <span className="text-xs text-muted-foreground/70">
        {currentStep} of {steps}
      </span>
      <RadialProgressBar steps={steps} currentStep={currentStep} />
    </div>
  );
};
