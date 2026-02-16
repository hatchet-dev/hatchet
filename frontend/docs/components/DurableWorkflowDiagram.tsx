import React, { useState, useEffect } from "react";

const DurableWorkflowDiagram: React.FC = () => {
  // Phases: 0=running, 1=checkpoint, 2=interrupted, 3=resumed, 4=complete
  const [phase, setPhase] = useState(0);

  useEffect(() => {
    const durations = [1500, 1200, 1800, 1200, 1500];
    const timer = setTimeout(() => {
      setPhase((prev) => (prev + 1) % 5);
    }, durations[phase]);
    return () => clearTimeout(timer);
  }, [phase]);

  const nodeW = 120;
  const nodeH = 44;
  const rx = 8;

  // Timeline positions
  const steps = [
    { x: 30, label: "Do Work", sub: "step 1" },
    { x: 175, label: "Checkpoint", sub: "save state" },
    { x: 320, label: "Interrupted", sub: "worker crash" },
    { x: 465, label: "Restore", sub: "new worker" },
    { x: 610, label: "Complete", sub: "step 2" },
  ];
  const y = 90;

  const phaseColors = [
    {
      fill: "rgba(49,46,129,0.3)",
      stroke: "rgb(99,102,241)",
      text: "#c7d2fe",
      sub: "#818cf8",
    },
    {
      fill: "rgba(120,53,15,0.25)",
      stroke: "rgb(245,158,11)",
      text: "#fcd34d",
      sub: "#d97706",
    },
    {
      fill: "rgba(127,29,29,0.25)",
      stroke: "rgb(239,68,68)",
      text: "#fca5a5",
      sub: "#ef4444",
    },
    {
      fill: "rgba(6,78,59,0.3)",
      stroke: "rgb(16,185,129)",
      text: "#a7f3d0",
      sub: "#6ee7b7",
    },
    {
      fill: "rgba(6,78,59,0.3)",
      stroke: "rgb(16,185,129)",
      text: "#a7f3d0",
      sub: "#6ee7b7",
    },
  ];

  const statusLabels = [
    "running...",
    "checkpointing...",
    "interrupted!",
    "restoring...",
    "complete!",
  ];
  const statusColors = ["#818cf8", "#fcd34d", "#fca5a5", "#6ee7b7", "#6ee7b7"];

  return (
    <div className="my-8 flex flex-col items-center gap-4">
      {/* Status bar */}
      <div className="flex items-center gap-3 rounded-lg border border-neutral-700/40 bg-neutral-900/50 px-4 py-2">
        <span className="text-xs font-medium text-gray-400">Durable task:</span>
        {steps.map((s, i) => (
          <span
            key={i}
            className="flex h-2 w-2 rounded-full transition-all duration-300"
            style={{
              backgroundColor:
                i < phase
                  ? i === 2
                    ? "rgb(239,68,68)"
                    : "rgb(16,185,129)"
                  : i === phase
                    ? phaseColors[i].stroke
                    : "#444",
              boxShadow:
                i === phase ? `0 0 6px ${phaseColors[i].stroke}` : "none",
            }}
          />
        ))}
        <span
          className="ml-1 text-xs font-medium transition-colors duration-300"
          style={{ color: statusColors[phase] }}
        >
          {statusLabels[phase]}
        </span>
      </div>

      {/* Diagram */}
      <div className="w-full max-w-3xl rounded-xl border border-neutral-700/40 bg-neutral-900/30 p-6 backdrop-blur-sm">
        <svg
          viewBox="0 0 760 200"
          className="w-full h-auto"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient id="dw-indigo" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="rgb(99,102,241)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(129,140,248)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="dw-amber" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="rgb(245,158,11)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(252,211,77)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="dw-red" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="rgb(239,68,68)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(248,113,113)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="dw-emerald" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="rgb(16,185,129)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(52,211,153)"
                stopOpacity="0.3"
              />
            </linearGradient>
          </defs>

          <style>{`
            @keyframes dw-dash {
              to { stroke-dashoffset: -20; }
            }
            .dw-flow {
              stroke-dasharray: 8 6;
              animation: dw-dash 0.8s linear infinite;
            }
            @keyframes dw-shake {
              0%, 100% { transform: translateX(0); }
              25% { transform: translateX(-2px); }
              75% { transform: translateX(2px); }
            }
          `}</style>

          {/* Timeline base line */}
          <line
            x1="30"
            y1={y + nodeH + 20}
            x2="730"
            y2={y + nodeH + 20}
            stroke="#333"
            strokeWidth="1"
          />

          {/* Progress line */}
          <line
            x1="30"
            y1={y + nodeH + 20}
            x2={steps[Math.min(phase, 4)].x + nodeW / 2}
            y2={y + nodeH + 20}
            stroke={
              phase === 2
                ? "rgb(239,68,68)"
                : phase >= 3
                  ? "rgb(16,185,129)"
                  : "rgb(99,102,241)"
            }
            strokeWidth="2"
            style={{ transition: "all 0.6s ease" }}
          />

          {/* Edges between nodes */}
          {steps.slice(0, -1).map((s, i) => {
            const nextX = steps[i + 1].x;
            const isActive = i === phase - 1 || (i < phase && phase > i + 1);
            const isCurrent = i === phase;

            let edgeColor = "#333";
            if (i < phase) {
              edgeColor =
                i === 1
                  ? "rgb(245,158,11)"
                  : i === 2
                    ? "rgb(16,185,129)"
                    : "rgb(99,102,241)";
            }
            if (isCurrent) {
              edgeColor = phaseColors[phase].stroke;
            }

            return (
              <path
                key={`edge-${i}`}
                d={`M ${s.x + nodeW + 2} ${y + nodeH / 2} L ${nextX - 2} ${y + nodeH / 2}`}
                fill="none"
                stroke={edgeColor}
                strokeWidth="2"
                className={isCurrent ? "dw-flow" : ""}
                style={{
                  opacity: i <= phase ? 1 : 0.15,
                  transition: "opacity 0.4s ease, stroke 0.4s ease",
                }}
              />
            );
          })}

          {/* Nodes */}
          {steps.map((s, i) => {
            const isActive = i === phase;
            const isPast = i < phase;
            const gradIds = [
              "dw-indigo",
              "dw-amber",
              "dw-red",
              "dw-emerald",
              "dw-emerald",
            ];

            let fill = "rgba(30,30,30,0.15)";
            let stroke = "#444";
            let textColor = "#666";
            let subColor = "#555";

            if (isActive) {
              fill = phaseColors[i].fill;
              stroke = phaseColors[i].stroke;
              textColor = phaseColors[i].text;
              subColor = phaseColors[i].sub;
            } else if (isPast) {
              fill = phaseColors[i].fill;
              stroke = phaseColors[i].stroke;
              textColor = phaseColors[i].text;
              subColor = phaseColors[i].sub;
              // Dim past nodes slightly
            }

            return (
              <g key={`node-${i}`}>
                <rect
                  x={s.x}
                  y={y}
                  width={nodeW}
                  height={nodeH}
                  rx={rx}
                  fill={fill}
                  stroke={stroke}
                  strokeWidth={isActive ? "2" : "1.5"}
                  strokeDasharray={
                    i === 1 && (isActive || isPast) ? "4 3" : "none"
                  }
                  style={{
                    transition: "all 0.4s ease",
                    opacity: isPast && !isActive ? 0.7 : i > phase ? 0.3 : 1,
                  }}
                />
                <text
                  x={s.x + nodeW / 2}
                  y={y + nodeH / 2 - 3}
                  textAnchor="middle"
                  fill={textColor}
                  fontSize="12"
                  fontWeight="500"
                  style={{ transition: "fill 0.4s ease" }}
                >
                  {s.label}
                </text>
                <text
                  x={s.x + nodeW / 2}
                  y={y + nodeH / 2 + 12}
                  textAnchor="middle"
                  fill={subColor}
                  fontSize="9"
                  style={{ transition: "fill 0.4s ease" }}
                >
                  {s.sub}
                </text>

                {/* Lightning bolt icon for crash */}
                {i === 2 && isActive && (
                  <text
                    x={s.x + nodeW / 2}
                    y={y - 10}
                    textAnchor="middle"
                    fill="#ef4444"
                    fontSize="16"
                  >
                    ⚡
                  </text>
                )}

                {/* Checkpoint icon */}
                {i === 1 && (isActive || isPast) && (
                  <text
                    x={s.x + nodeW / 2}
                    y={y - 10}
                    textAnchor="middle"
                    fill="#fcd34d"
                    fontSize="14"
                  >
                    💾
                  </text>
                )}

                {/* Timeline dot */}
                <circle
                  cx={s.x + nodeW / 2}
                  cy={y + nodeH + 20}
                  r={i <= phase ? 4 : 3}
                  fill={
                    i < phase
                      ? i === 2
                        ? "rgb(239,68,68)"
                        : "rgb(16,185,129)"
                      : i === phase
                        ? phaseColors[i].stroke
                        : "#444"
                  }
                  style={{ transition: "all 0.4s ease" }}
                />
              </g>
            );
          })}

          {/* Arrow showing "skip replay" from checkpoint to restore */}
          {phase >= 3 && (
            <g
              style={{
                opacity: phase >= 3 ? 1 : 0,
                transition: "opacity 0.5s ease",
              }}
            >
              <path
                d={`M ${steps[1].x + nodeW / 2} ${y + nodeH + 34} C ${steps[1].x + nodeW / 2} ${y + nodeH + 60}, ${steps[3].x + nodeW / 2} ${y + nodeH + 60}, ${steps[3].x + nodeW / 2} ${y + nodeH + 34}`}
                fill="none"
                stroke="rgb(16,185,129)"
                strokeWidth="1.5"
                strokeDasharray="4 3"
                markerEnd=""
              />
              <text
                x={(steps[1].x + steps[3].x) / 2 + nodeW / 2}
                y={y + nodeH + 62}
                textAnchor="middle"
                fill="#6ee7b7"
                fontSize="9"
              >
                replay from checkpoint
              </text>
            </g>
          )}
        </svg>
      </div>
    </div>
  );
};

export default DurableWorkflowDiagram;
