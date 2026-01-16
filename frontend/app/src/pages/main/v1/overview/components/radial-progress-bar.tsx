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
  const strokeWidth = Math.min(Math.max(6 * scale, 1.5), 2.5);

  // Absolute gap size in pixels (scaled) - this stays constant regardless of step count
  const gapSize = Math.min(16 * scale, 6); // Gap size in pixels
  const gapAngle = gapSize / radius; // Convert to angle in radians

  // Function to create an arc path for a single segment
  const createArcPath = (startAngle: number, endAngle: number): string => {
    const startX = center + radius * Math.cos(startAngle);
    const startY = center + radius * Math.sin(startAngle);
    const endX = center + radius * Math.cos(endAngle);
    const endY = center + radius * Math.sin(endAngle);

    const largeArcFlag = endAngle - startAngle > Math.PI ? 1 : 0;

    return `M ${startX} ${startY} A ${radius} ${radius} 0 ${largeArcFlag} 1 ${endX} ${endY}`;
  };

  // Generate arc segments for each step
  const arcSegments = Array.from({ length: steps }, (_, i) => {
    // Calculate angles for this segment
    const stepAngle = (2 * Math.PI) / steps;
    const segmentAngle = stepAngle - gapAngle;

    // Center the segment within its allocated stepAngle space
    // This ensures symmetry and prevents rotation offset
    // Start angle: beginning of step + half the gap (to center the segment)
    const startAngle = i * stepAngle - Math.PI / 2 + gapAngle / 2;
    // End angle: start + segment length
    const endAngle = startAngle + segmentAngle;

    const isCompleted = i < currentStep;
    const isCurrent = i === currentStep - 1;

    return {
      path: createArcPath(startAngle, endAngle),
      isCompleted,
      isCurrent,
      index: i,
    };
  });

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      className="relative"
    >
      {/* Background circle - divided into segments (only show unfilled segments) */}
      {arcSegments.map((segment, index) => {
        // Hide background segment if it's filled in the progress layer
        if (segment.isCompleted || segment.isCurrent) {
          return null;
        }

        return (
          <path
            key={`bg-${index}`}
            d={segment.path}
            fill="none"
            stroke="hsl(var(--border))"
            strokeWidth={strokeWidth}
            strokeLinecap="round"
          />
        );
      })}

      {/* Progress segments - only show completed and current steps */}
      {arcSegments.map((segment, index) => {
        if (!segment.isCompleted && !segment.isCurrent) {
          return null;
        }

        return (
          <path
            key={`progress-${index}`}
            d={segment.path}
            fill="none"
            stroke="hsl(var(--brand))"
            strokeWidth={strokeWidth}
            strokeLinecap="round"
            className="transition-all duration-300 ease-out"
          />
        );
      })}
    </svg>
  );
};
