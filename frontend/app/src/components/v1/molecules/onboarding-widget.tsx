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
  const circumference = 2 * Math.PI * 54; // ≈ 339.292
  const progress = currentStep / steps;
  const strokeDashoffset = circumference * (1 - progress);

  // Calculate step positions around the circle
  const centerX = 60;
  const centerY = 60;
  const radius = 54;
  const stepMarkers = Array.from({ length: steps }, (_, i) => {
    const angle = (i / steps) * 2 * Math.PI - Math.PI / 2; // Start from top (-90 degrees)
    const x = centerX + radius * Math.cos(angle);
    const y = centerY + radius * Math.sin(angle);
    const isCompleted = i < currentStep;
    const isCurrent = i === currentStep - 1;
    return { x, y, isCompleted, isCurrent, stepNumber: i + 1 };
  });

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
      <svg width="120" height="120" viewBox="0 0 120 120" className="relative">
        {/* Background circle */}
        <circle
          cx="60"
          cy="60"
          r="54"
          fill="none"
          stroke="hsl(var(--border))"
          strokeWidth="12"
        />
        {/* Progress circle */}
        <circle
          cx="60"
          cy="60"
          r="54"
          fill="none"
          stroke="hsl(var(--brand))"
          strokeWidth="12"
          strokeDasharray={circumference}
          strokeDashoffset={strokeDashoffset}
          strokeLinecap="round"
          transform="rotate(-90 60 60)"
          className="transition-all duration-300 ease-out"
        />
        {/* Step markers */}
        {stepMarkers.map((marker, index) => (
          <g key={index}>
            {/* Step dot */}
            <circle
              cx={marker.x}
              cy={marker.y}
              r={marker.isCurrent ? 6 : marker.isCompleted ? 5 : 4}
              fill={
                marker.isCurrent
                  ? 'hsl(var(--brand))'
                  : marker.isCompleted
                    ? 'hsl(var(--brand))'
                    : 'hsla(var(--brand), 0.5)'
              }
              className="transition-all duration-300"
            />
            {/* Step number */}
            {/* <text
              x={marker.x}
              y={marker.y}
              textAnchor="middle"
              dominantBaseline="central"
              className="text-[10px] font-semibold fill-white pointer-events-none"
              style={{ fontSize: '10px' }}
            >
              {marker.isCompleted || marker.isCurrent ? marker.stepNumber : ''}
            </text> */}
          </g>
        ))}
      </svg>
    </div>
  );
};
