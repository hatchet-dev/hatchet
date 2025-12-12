import { Badge } from '@/components/v1/ui/badge';

export function StepProgress<T = unknown>({
  steps,
  currentStep,
  onStepClick,
}: {
  steps: Array<T>;
  currentStep: number;
  onStepClick?: (stepIndex: number) => void;
}) {
  return (
    <div className="flex flex-col space-y-2 text-center">
      <div className="mb-6 flex items-center justify-center">
        {steps.map((_, index) => (
          <div key={index} className="flex items-center">
            <Badge
              variant={index <= currentStep ? 'default' : 'secondary'}
              className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium transition-colors ${
                index <= currentStep
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground'
              } ${
                onStepClick
                  ? 'cursor-pointer hover:scale-105 hover:opacity-80 active:scale-95'
                  : ''
              }`}
              onClick={() => onStepClick?.(index)}
            >
              {index + 1}
            </Badge>
            {index < steps.length - 1 && (
              <div
                className={`mx-3 h-0.5 w-8 transition-colors ${
                  index < currentStep ? 'bg-primary' : 'bg-muted'
                }`}
              />
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
