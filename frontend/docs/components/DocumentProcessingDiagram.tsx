import React, { useState, useEffect } from "react";

const STAGES = [
  { id: "ingest", label: "Ingest", color: "#818cf8" },
  { id: "parse", label: "Parse", color: "#38bdf8" },
  { id: "extract", label: "Extract", color: "#fbbf24" },
  { id: "validate", label: "Validate", color: "#a78bfa" },
  { id: "output", label: "Output", color: "#34d399" },
] as const;

type StageId = (typeof STAGES)[number]["id"];

const StageIcon: React.FC<{ id: StageId; color: string; size?: number }> = ({
  id,
  color,
  size = 16,
}) => {
  const props = {
    width: size,
    height: size,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: color,
    strokeWidth: "2",
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
  };

  switch (id) {
    case "ingest":
      return (
        <svg {...props}>
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="7 10 12 15 17 10" />
          <line x1="12" y1="15" x2="12" y2="3" />
        </svg>
      );
    case "parse":
      return (
        <svg {...props}>
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
          <polyline points="14 2 14 8 20 8" />
          <line x1="16" y1="13" x2="8" y2="13" />
          <line x1="16" y1="17" x2="8" y2="17" />
          <line x1="10" y1="9" x2="8" y2="9" />
        </svg>
      );
    case "extract":
      return (
        <svg {...props}>
          <path d="M12 3v18" />
          <path d="M8 7l4-4 4 4" />
          <path d="M8 17l4 4 4-4" />
          <circle cx="12" cy="12" r="3" />
        </svg>
      );
    case "validate":
      return (
        <svg {...props}>
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
          <polyline points="22 4 12 14.01 9 11.01" />
        </svg>
      );
    case "output":
      return (
        <svg {...props}>
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="17 8 12 3 7 8" />
          <line x1="12" y1="3" x2="12" y2="15" />
        </svg>
      );
  }
};

const DocumentProcessingDiagram: React.FC = () => {
  const [activeStage, setActiveStage] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setActiveStage((prev) => (prev + 1) % STAGES.length);
    }, 1500);
    return () => clearInterval(interval);
  }, []);

  const stageWidth = 64;
  const gap = 12;
  const totalWidth = STAGES.length * stageWidth + (STAGES.length - 1) * gap;
  const startX = (480 - totalWidth) / 2;

  return (
    <div
      className="my-6 rounded-xl border p-6"
      style={{
        borderColor: "rgba(99,102,241,0.2)",
        backgroundColor: "rgba(49,46,129,0.04)",
      }}
    >
      <svg
        viewBox="0 0 480 160"
        className="mx-auto w-full"
        style={{ maxWidth: 550 }}
      >
        {/* Connecting arrows */}
        {STAGES.slice(0, -1).map((_, i) => {
          const fromX = startX + i * (stageWidth + gap) + stageWidth;
          const toX = startX + (i + 1) * (stageWidth + gap);
          const y = 70;
          const isActive = i === activeStage || i + 1 === activeStage;

          return (
            <g key={`arrow-${i}`}>
              <line
                x1={fromX + 2}
                y1={y}
                x2={toX - 2}
                y2={y}
                stroke={isActive ? STAGES[i].color : "#374151"}
                strokeWidth={isActive ? 2 : 1.5}
                opacity={isActive ? 1 : 0.4}
                style={{ transition: "all 0.5s ease" }}
              />
              <polygon
                points={`${toX - 2},${y} ${toX - 8},${y - 4} ${toX - 8},${y + 4}`}
                fill={isActive ? STAGES[i + 1].color : "#374151"}
                opacity={isActive ? 1 : 0.4}
                style={{ transition: "all 0.5s ease" }}
              />
              {i === activeStage && activeStage < STAGES.length - 1 && (
                <circle r="3" fill={STAGES[i].color}>
                  <animateMotion
                    dur="1.5s"
                    repeatCount="1"
                    path={`M ${fromX + 2} ${y} L ${toX - 2} ${y}`}
                  />
                </circle>
              )}
            </g>
          );
        })}

        {/* Stage boxes */}
        {STAGES.map((stage, i) => {
          const x = startX + i * (stageWidth + gap);
          const y = 40;
          const isActive = i === activeStage;

          return (
            <g key={stage.id}>
              {isActive && (
                <rect
                  x={x - 4}
                  y={y - 4}
                  width={stageWidth + 8}
                  height={68}
                  rx={14}
                  fill={stage.color}
                  opacity={0.1}
                >
                  <animate
                    attributeName="opacity"
                    values="0.1;0.05;0.1"
                    dur="1.5s"
                    repeatCount="indefinite"
                  />
                </rect>
              )}
              <rect
                x={x}
                y={y}
                width={stageWidth}
                height={60}
                rx={10}
                fill={isActive ? `${stage.color}15` : "#1f2937"}
                stroke={isActive ? stage.color : "#374151"}
                strokeWidth={isActive ? 2 : 1}
                style={{ transition: "all 0.5s ease" }}
              />
              <foreignObject
                x={x + stageWidth / 2 - 8}
                y={y + 12}
                width={16}
                height={16}
              >
                <StageIcon
                  id={stage.id}
                  color={isActive ? stage.color : "#6b7280"}
                />
              </foreignObject>
              <text
                x={x + stageWidth / 2}
                y={y + 48}
                textAnchor="middle"
                fontSize="10"
                fontWeight={isActive ? 600 : 400}
                fill={isActive ? stage.color : "#9ca3af"}
                style={{ transition: "all 0.5s ease" }}
              >
                {stage.label}
              </text>
            </g>
          );
        })}

        {/* Per-file fanout under Parse */}
        <g
          opacity={activeStage === 1 ? 1 : 0.3}
          style={{ transition: "opacity 0.5s ease" }}
        >
          <text
            x={startX + 1 * (stageWidth + gap) + stageWidth / 2}
            y={125}
            textAnchor="middle"
            fontSize="9"
            fill="#38bdf8"
          >
            per-file fanout
          </text>
          <line
            x1={startX + 1 * (stageWidth + gap) + stageWidth / 2}
            y1={102}
            x2={startX + 1 * (stageWidth + gap) + stageWidth / 2}
            y2={118}
            stroke="#38bdf8"
            strokeWidth={1}
            strokeDasharray="3 2"
          />
        </g>

        {/* Rate-limited under Extract */}
        <g
          opacity={activeStage === 2 ? 1 : 0.3}
          style={{ transition: "opacity 0.5s ease" }}
        >
          <text
            x={startX + 2 * (stageWidth + gap) + stageWidth / 2}
            y={125}
            textAnchor="middle"
            fontSize="9"
            fill="#fbbf24"
          >
            rate-limited
          </text>
          <line
            x1={startX + 2 * (stageWidth + gap) + stageWidth / 2}
            y1={102}
            x2={startX + 2 * (stageWidth + gap) + stageWidth / 2}
            y2={118}
            stroke="#fbbf24"
            strokeWidth={1}
            strokeDasharray="3 2"
          />
        </g>

        {/* Retry indicator under Validate */}
        <g
          opacity={activeStage === 3 ? 1 : 0.3}
          style={{ transition: "opacity 0.5s ease" }}
        >
          <text
            x={startX + 3 * (stageWidth + gap) + stageWidth / 2}
            y={125}
            textAnchor="middle"
            fontSize="9"
            fill="#a78bfa"
          >
            retries on failure
          </text>
          <line
            x1={startX + 3 * (stageWidth + gap) + stageWidth / 2}
            y1={102}
            x2={startX + 3 * (stageWidth + gap) + stageWidth / 2}
            y2={118}
            stroke="#a78bfa"
            strokeWidth={1}
            strokeDasharray="3 2"
          />
        </g>
      </svg>

      <div className="mt-3 flex items-center justify-center gap-2">
        {STAGES.map((stage, i) => (
          <div
            key={stage.id}
            className="h-1.5 rounded-full"
            style={{
              width: i === activeStage ? 32 : 12,
              backgroundColor:
                i === activeStage ? stage.color : "rgba(55,65,81,0.5)",
              transition: "all 0.5s ease",
            }}
          />
        ))}
      </div>
    </div>
  );
};

export default DocumentProcessingDiagram;
