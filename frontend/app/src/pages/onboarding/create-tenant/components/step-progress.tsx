import { Badge } from '@/components/ui/badge';

export function StepProgress({
  steps,
  currentStep,
  onStepClick,
}: {
  steps: any[];
  currentStep: number;
  onStepClick?: (stepIndex: number) => void;
}) {
  return (
    <div className="flex flex-col space-y-2 text-center">
      <div className="flex justify-center items-center mb-6">
        {steps.map((_, index) => (
          <div key={index} className="flex items-center">
            <Badge
              variant={index <= currentStep ? 'default' : 'secondary'}
              className={`h-8 w-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors ${
                index <= currentStep
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground'
              } ${
                onStepClick
                  ? 'cursor-pointer hover:opacity-80 hover:scale-105 active:scale-95'
                  : ''
              }`}
              onClick={() => onStepClick?.(index)}
            >
              {index + 1}
            </Badge>
            {index < steps.length - 1 && (
              <div
                className={`w-8 h-0.5 mx-3 transition-colors ${
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
