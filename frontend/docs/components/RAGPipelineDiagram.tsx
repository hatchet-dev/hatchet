import React, { useState, useEffect } from "react";

const STAGES = [
  { id: "ingest", label: "Ingest", icon: "📥", color: "#818cf8" },
  { id: "chunk", label: "Chunk", icon: "✂️", color: "#38bdf8" },
  { id: "embed", label: "Embed", icon: "🧮", color: "#fbbf24" },
  { id: "index", label: "Index", icon: "🗃️", color: "#34d399" },
  { id: "query", label: "Query", icon: "🔍", color: "#f472b6" },
] as const;

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
        borderColor: "rgba(99,102,241,0.2)",
        backgroundColor: "rgba(49,46,129,0.04)",
      }}
    >
      <svg
        viewBox="0 0 440 160"
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
              {/* Animated dot traveling along the arrow */}
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
                fill={isActive ? `${stage.color}15` : "#1f2937"}
                stroke={isActive ? stage.color : "#374151"}
                strokeWidth={isActive ? 2 : 1}
                style={{ transition: "all 0.5s ease" }}
              />
              {/* Icon */}
              <text
                x={x + stageWidth / 2}
                y={y + 28}
                textAnchor="middle"
                fontSize="18"
              >
                {stage.icon}
              </text>
              {/* Label */}
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

        {/* Fan-out indicator under chunk stage */}
        <g opacity={activeStage === 1 ? 1 : 0.3} style={{ transition: "opacity 0.5s ease" }}>
          <text
            x={startX + 1 * (stageWidth + gap) + stageWidth / 2}
            y={125}
            textAnchor="middle"
            fontSize="9"
            fill="#38bdf8"
          >
            fan-out to N chunks
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

        {/* Rate limit indicator under embed stage */}
        <g opacity={activeStage === 2 ? 1 : 0.3} style={{ transition: "opacity 0.5s ease" }}>
          <text
            x={startX + 2 * (stageWidth + gap) + stageWidth / 2}
            y={125}
            textAnchor="middle"
            fontSize="9"
            fill="#fbbf24"
          >
            rate-limited API
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

        {/* Retry indicator under index stage */}
        <g opacity={activeStage === 3 ? 1 : 0.3} style={{ transition: "opacity 0.5s ease" }}>
          <text
            x={startX + 3 * (stageWidth + gap) + stageWidth / 2}
            y={125}
            textAnchor="middle"
            fontSize="9"
            fill="#34d399"
          >
            retries on failure
          </text>
          <line
            x1={startX + 3 * (stageWidth + gap) + stageWidth / 2}
            y1={102}
            x2={startX + 3 * (stageWidth + gap) + stageWidth / 2}
            y2={118}
            stroke="#34d399"
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
                i === activeStage ? stage.color : "rgba(55,65,81,0.5)",
              transition: "all 0.5s ease",
            }}
          />
        ))}
      </div>
    </div>
  );
};

export default RAGPipelineDiagram;
