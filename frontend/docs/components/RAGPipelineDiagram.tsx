import React, { useState, useEffect } from "react";
import { brand, state, inactive, container } from "./diagram-colors";

const STAGES = [
  { id: "ingest", label: "Ingest", color: "#3392FF" },
  { id: "chunk", label: "Chunk", color: "#BC46DD" },
  { id: "embed", label: "Embed", color: "#EAB308" },
  { id: "index", label: "Index", color: "#22C55E" },
  { id: "query", label: "Query", color: "#B8D41C" },
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
      // Download/import arrow
      return (
        <svg {...props}>
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="7 10 12 15 17 10" />
          <line x1="12" y1="15" x2="12" y2="3" />
        </svg>
      );
    case "chunk":
      // Scissors
      return (
        <svg {...props}>
          <circle cx="6" cy="6" r="3" />
          <circle cx="6" cy="18" r="3" />
          <line x1="20" y1="4" x2="8.12" y2="15.88" />
          <line x1="14.47" y1="14.48" x2="20" y2="20" />
          <line x1="8.12" y1="8.12" x2="12" y2="12" />
        </svg>
      );
    case "embed":
      // Grid/vector
      return (
        <svg {...props}>
          <rect x="3" y="3" width="7" height="7" />
          <rect x="14" y="3" width="7" height="7" />
          <rect x="14" y="14" width="7" height="7" />
          <rect x="3" y="14" width="7" height="7" />
        </svg>
      );
    case "index":
      // Database
      return (
        <svg {...props}>
          <ellipse cx="12" cy="5" rx="9" ry="3" />
          <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" />
          <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" />
        </svg>
      );
    case "query":
      // Search
      return (
        <svg {...props}>
          <circle cx="11" cy="11" r="8" />
          <line x1="21" y1="21" x2="16.65" y2="16.65" />
        </svg>
      );
  }
};

const RAGPipelineDiagram: React.FC = () => {
  const [activeStage, setActiveStage] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setActiveStage((prev) => (prev + 1) % STAGES.length);
    }, 1500);
    return () => clearInterval(interval);
  }, []);

  const stageWidth = 72;
  const gap = 16;
  const totalWidth = STAGES.length * stageWidth + (STAGES.length - 1) * gap;
  const startX = (440 - totalWidth) / 2;

  return (
    <div
      className="my-6 rounded-xl border p-6"
      style={{
        borderColor: "rgba(51, 146, 255, 0.2)",
        backgroundColor: "rgba(10, 16, 41, 0.04)",
      }}
    >
      <svg
        viewBox="0 0 440 160"
        className="mx-auto w-full"
        style={{ maxWidth: 550 }}
        overflow="hidden"
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
                stroke={isActive ? STAGES[i].color : "#1C2B4A"}
                strokeWidth={isActive ? 2 : 1.5}
                opacity={isActive ? 1 : 0.4}
                style={{ transition: "all 0.5s ease" }}
              />
              <polygon
                points={`${toX - 2},${y} ${toX - 8},${y - 4} ${toX - 8},${y + 4}`}
                fill={isActive ? STAGES[i].color : "#1C2B4A"}
                opacity={isActive ? 1 : 0.4}
                style={{ transition: "all 0.5s ease" }}
              />
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
              {/* Glow */}
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
              {/* Box */}
              <rect
                x={x}
                y={y}
                width={stageWidth}
                height={60}
                rx={10}
                fill={isActive ? `${stage.color}15` : "#0A1029"}
                stroke={isActive ? stage.color : "#1C2B4A"}
                strokeWidth={isActive ? 2 : 1}
                style={{ transition: "all 0.5s ease" }}
              />
              {/* Icon */}
              <foreignObject
                x={x + stageWidth / 2 - 8}
                y={y + 12}
                width={16}
                height={16}
              >
                <StageIcon
                  id={stage.id}
                  color={isActive ? stage.color : "#64748B"}
                />
              </foreignObject>
              {/* Label */}
              <text
                x={x + stageWidth / 2}
                y={y + 48}
                textAnchor="middle"
                fontSize="10"
                fontWeight={isActive ? 600 : 400}
                fill={isActive ? stage.color : "#A5C5E9"}
                style={{ transition: "all 0.5s ease" }}
              >
                {stage.label}
              </text>
            </g>
          );
        })}

        {/* Fan-out indicator under chunk stage */}
        <g
          opacity={activeStage === 1 ? 1 : 0.3}
          style={{ transition: "opacity 0.5s ease" }}
        >
          <text
            x={startX + 1 * (stageWidth + gap) + stageWidth / 2}
            y={125}
            textAnchor="middle"
            fontSize="9"
            fill="#BC46DD"
          >
            fan-out to N chunks
          </text>
          <line
            x1={startX + 1 * (stageWidth + gap) + stageWidth / 2}
            y1={102}
            x2={startX + 1 * (stageWidth + gap) + stageWidth / 2}
            y2={118}
            stroke="#BC46DD"
            strokeWidth={1}
            strokeDasharray="3 2"
          />
        </g>

        {/* Rate limit indicator under embed stage */}
        <g
          opacity={activeStage === 2 ? 1 : 0.3}
          style={{ transition: "opacity 0.5s ease" }}
        >
          <text
            x={startX + 2 * (stageWidth + gap) + stageWidth / 2}
            y={125}
            textAnchor="middle"
            fontSize="9"
            fill="#EAB308"
          >
            rate-limited API
          </text>
          <line
            x1={startX + 2 * (stageWidth + gap) + stageWidth / 2}
            y1={102}
            x2={startX + 2 * (stageWidth + gap) + stageWidth / 2}
            y2={118}
            stroke="#EAB308"
            strokeWidth={1}
            strokeDasharray="3 2"
          />
        </g>

        {/* Retry indicator under index stage */}
        <g
          opacity={activeStage === 3 ? 1 : 0.3}
          style={{ transition: "opacity 0.5s ease" }}
        >
          <text
            x={startX + 3 * (stageWidth + gap) + stageWidth / 2}
            y={125}
            textAnchor="middle"
            fontSize="9"
            fill="#22C55E"
          >
            retries on failure
          </text>
          <line
            x1={startX + 3 * (stageWidth + gap) + stageWidth / 2}
            y1={102}
            x2={startX + 3 * (stageWidth + gap) + stageWidth / 2}
            y2={118}
            stroke="#22C55E"
            strokeWidth={1}
            strokeDasharray="3 2"
          />
        </g>
      </svg>

      {/* Progress indicator */}
      <div className="mt-3 flex items-center justify-center gap-2">
        {STAGES.map((stage, i) => (
          <div
            key={stage.id}
            className="h-1.5 rounded-full"
            style={{
              width: i === activeStage ? 32 : 12,
              backgroundColor:
                i === activeStage ? stage.color : "rgba(10, 16, 41, 0.5)",
              transition: "all 0.5s ease",
            }}
          />
        ))}
      </div>
    </div>
  );
};

export default RAGPipelineDiagram;
