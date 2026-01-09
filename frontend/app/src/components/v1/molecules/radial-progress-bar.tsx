interface RadialProgressBarProps {
  steps: number;
  currentStep: number;
  size?: number;
}

export const RadialProgressBar = ({
  steps,
  currentStep,
  size = 32,
}: RadialProgressBarProps) => {
  // Scale all dimensions based on size (original design was 120px)
  const scale = size / 120;
  const center = size / 2;
  const radius = 54 * scale;
  const strokeWidth = 12 * scale;
  const circumference = 2 * Math.PI * radius;
  const progress = currentStep / steps;
  const strokeDashoffset = circumference * (1 - progress);

  // Calculate step positions around the circle
  const stepMarkers = Array.from({ length: steps }, (_, i) => {
    const angle = (i / steps) * 2 * Math.PI - Math.PI / 2; // Start from top (-90 degrees)
    const x = center + radius * Math.cos(angle);
    const y = center + radius * Math.sin(angle);
    const isCompleted = i < currentStep;
    const isCurrent = i === currentStep - 1;
    return { x, y, isCompleted, isCurrent, stepNumber: i + 1 };
  });

  // Scale marker dot sizes
  const markerRadiusCurrent = 6 * scale;
  const markerRadiusCompleted = 5 * scale;
  const markerRadiusUpcoming = 4 * scale;

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      className="relative"
    >
      {/* Background circle */}
      <circle
        cx={center}
        cy={center}
        r={radius}
        fill="none"
        stroke="hsl(var(--border))"
        strokeWidth={strokeWidth}
      />
      {/* Progress circle */}
      <circle
        cx={center}
        cy={center}
        r={radius}
        fill="none"
        stroke="hsl(var(--brand))"
        strokeWidth={strokeWidth}
        strokeDasharray={circumference}
        strokeDashoffset={strokeDashoffset}
        strokeLinecap="round"
        transform={`rotate(-90 ${center} ${center})`}
        className="transition-all duration-300 ease-out"
      />
      {/* Step markers */}
      {stepMarkers.map((marker, index) => (
        <g key={index}>
          {/* Step dot */}
          <circle
            cx={marker.x}
            cy={marker.y}
            r={
              marker.isCurrent
                ? markerRadiusCurrent
                : marker.isCompleted
                  ? markerRadiusCompleted
                  : markerRadiusUpcoming
            }
            fill={
              marker.isCurrent
                ? 'hsl(var(--brand))'
                : marker.isCompleted
                  ? 'hsl(var(--brand))'
                  : 'hsla(var(--brand), 0.5)'
            }
            className="transition-all duration-300"
          />
        </g>
      ))}
    </svg>
  );
};
