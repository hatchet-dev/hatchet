import React, { useState, useEffect } from "react";

const ITEMS = Array.from({ length: 8 }, (_, i) => i);

/** SVG checkmark for completed items */
const CheckIcon: React.FC<{ x: number; y: number; color: string }> = ({
  x,
  y,
  color,
}) => (
  <g transform={`translate(${x - 6}, ${y - 6})`}>
    <svg
      width="12"
      height="12"
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth="3"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <polyline points="20 6 9 17 4 12" />
    </svg>
  </g>
);

const BatchProcessingDiagram: React.FC = () => {
  const [processedCount, setProcessedCount] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setProcessedCount((prev) => (prev >= ITEMS.length ? 0 : prev + 1));
    }, 600);
    return () => clearInterval(interval);
  }, []);

  const colors = {
    pending: "#374151",
    processing: "#fbbf24",
    done: "#34d399",
  };

  return (
    <div
      className="my-6 rounded-xl border p-6"
      style={{
        borderColor: "rgba(99,102,241,0.2)",
        backgroundColor: "rgba(49,46,129,0.04)",
      }}
    >
      <svg
        viewBox="0 0 440 170"
        className="mx-auto w-full"
        style={{ maxWidth: 520 }}
      >
        {/* Trigger box */}
        <rect
          x={15}
          y={55}
          width={70}
          height={45}
          rx={8}
          fill="rgba(99,102,241,0.1)"
          stroke="#818cf8"
          strokeWidth={1.5}
        />
        {/* List icon */}
        <g transform="translate(43, 65)">
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#818cf8"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <line x1="8" y1="6" x2="21" y2="6" />
            <line x1="8" y1="12" x2="21" y2="12" />
            <line x1="8" y1="18" x2="21" y2="18" />
            <line x1="3" y1="6" x2="3.01" y2="6" />
            <line x1="3" y1="12" x2="3.01" y2="12" />
            <line x1="3" y1="18" x2="3.01" y2="18" />
          </svg>
        </g>
        <text
          x={50}
          y={89}
          textAnchor="middle"
          fontSize="9"
          fill="#818cf8"
          fontWeight={600}
        >
          Batch Input
        </text>

        {/* Arrow from trigger to items */}
        <line
          x1={87}
          y1={78}
          x2={110}
          y2={78}
          stroke="#818cf8"
          strokeWidth={1.5}
        />
        <polygon points="110,78 104,74 104,82" fill="#818cf8" />

        {/* Fan-out items grid (2 rows x 4 cols) */}
        {ITEMS.map((item) => {
          const col = item % 4;
          const row = Math.floor(item / 4);
          const x = 120 + col * 52;
          const y = 35 + row * 55;

          let status: "pending" | "processing" | "done";
          if (item < processedCount) {
            status = "done";
          } else if (item === processedCount) {
            status = "processing";
          } else {
            status = "pending";
          }

          const color = colors[status];

          return (
            <g key={item}>
              {status === "processing" && (
                <rect
                  x={x - 2}
                  y={y - 2}
                  width={44}
                  height={44}
                  rx={10}
                  fill={color}
                  opacity={0.15}
                >
                  <animate
                    attributeName="opacity"
                    values="0.15;0.08;0.15"
                    dur="0.6s"
                    repeatCount="indefinite"
                  />
                </rect>
              )}
              <rect
                x={x}
                y={y}
                width={40}
                height={40}
                rx={8}
                fill={
                  status === "done"
                    ? `${color}15`
                    : status === "processing"
                      ? `${color}15`
                      : "#1f2937"
                }
                stroke={color}
                strokeWidth={status === "pending" ? 1 : 2}
                style={{ transition: "all 0.3s ease" }}
              />
              {/* Status indicator */}
              {status === "done" ? (
                <CheckIcon x={x + 20} y={y + 16} color={color} />
              ) : status === "processing" ? (
                // Spinning indicator
                <g transform={`translate(${x + 14}, ${y + 10})`}>
                  <svg
                    width="12"
                    height="12"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke={color}
                    strokeWidth="3"
                    strokeLinecap="round"
                  >
                    <path d="M21 12a9 9 0 1 1-6.219-8.56">
                      <animateTransform
                        attributeName="transform"
                        type="rotate"
                        from="0 12 12"
                        to="360 12 12"
                        dur="1s"
                        repeatCount="indefinite"
                      />
                    </path>
                  </svg>
                </g>
              ) : (
                // File icon for pending
                <g transform={`translate(${x + 14}, ${y + 10})`}>
                  <svg
                    width="12"
                    height="12"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke={color}
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                    <polyline points="14 2 14 8 20 8" />
                  </svg>
                </g>
              )}
              <text
                x={x + 20}
                y={y + 33}
                textAnchor="middle"
                fontSize="8"
                fill={color}
                style={{ transition: "all 0.3s ease" }}
              >
                Item {item + 1}
              </text>
            </g>
          );
        })}

        {/* Arrow from items to results */}
        <line
          x1={332}
          y1={78}
          x2={355}
          y2={78}
          stroke="#34d399"
          strokeWidth={1.5}
        />
        <polygon points="355,78 349,74 349,82" fill="#34d399" />

        {/* Results box */}
        <rect
          x={360}
          y={55}
          width={70}
          height={45}
          rx={8}
          fill="rgba(52,211,153,0.1)"
          stroke="#34d399"
          strokeWidth={1.5}
        />
        {/* Bar chart icon */}
        <g transform="translate(388, 65)">
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#34d399"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <line x1="18" y1="20" x2="18" y2="10" />
            <line x1="12" y1="20" x2="12" y2="4" />
            <line x1="6" y1="20" x2="6" y2="14" />
          </svg>
        </g>
        <text
          x={395}
          y={89}
          textAnchor="middle"
          fontSize="9"
          fill="#34d399"
          fontWeight={600}
        >
          Results
        </text>
      </svg>

      {/* Progress bar */}
      <div className="mx-auto mt-3 flex max-w-xs items-center gap-3">
        <div
          className="h-2 flex-1 overflow-hidden rounded-full"
          style={{ backgroundColor: "rgba(55,65,81,0.5)" }}
        >
          <div
            className="h-full rounded-full"
            style={{
              width: `${(processedCount / ITEMS.length) * 100}%`,
              backgroundColor:
                processedCount === ITEMS.length ? "#34d399" : "#fbbf24",
              transition: "all 0.3s ease",
            }}
          />
        </div>
        <span className="text-xs tabular-nums" style={{ color: "#9ca3af" }}>
          {processedCount}/{ITEMS.length}
        </span>
      </div>
    </div>
  );
};

export default BatchProcessingDiagram;
