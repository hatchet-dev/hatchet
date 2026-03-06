import React, { useState, useEffect } from "react";
import { brand, state, fill, inactive, gradient } from "./diagram-colors";

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
      fill: fill.activeNode,
      stroke: brand.blue,
      text: brand.cyan,
      sub: brand.blue,
    },
    {
      fill: fill.running,
      stroke: state.running,
      text: state.runningLight,
      sub: state.runningDark,
    },
    {
      fill: fill.failed,
      stroke: state.failed,
      text: state.failedLight,
      sub: state.failed,
    },
    {
      fill: fill.success,
      stroke: state.success,
      text: state.successLighter,
      sub: state.successLight,
    },
    {
      fill: fill.success,
      stroke: state.success,
      text: state.successLighter,
      sub: state.successLight,
    },
  ];

  const statusLabels = [
    "running...",
    "checkpointing...",
    "interrupted!",
    "restoring...",
    "complete!",
  ];
  const statusColors = [
    brand.blue,
    state.runningLight,
    state.failedLight,
    state.successLight,
    state.successLight,
  ];

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
                    ? state.failed
                    : state.success
                  : i === phase
                    ? phaseColors[i].stroke
                    : inactive.dot,
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
            <linearGradient id="dw-blue" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop
                offset="0%"
                stopColor={gradient.blue[0]}
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor={gradient.blue[1]}
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="dw-yellow" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop
                offset="0%"
                stopColor={gradient.yellow[0]}
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor={gradient.yellow[1]}
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="dw-red" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor={gradient.red[0]} stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor={gradient.red[1]}
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="dw-green" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop
                offset="0%"
                stopColor={gradient.green[0]}
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor={gradient.green[1]}
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
          `}</style>

          {/* Timeline base line */}
          <line
            x1="30"
            y1={y + nodeH + 20}
            x2="730"
            y2={y + nodeH + 20}
            stroke={inactive.edge}
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
                ? state.failed
                : phase >= 3
                  ? state.success
                  : brand.blue
            }
            strokeWidth="2"
            style={{ transition: "all 0.6s ease" }}
          />

          {/* Edges between nodes */}
          {steps.slice(0, -1).map((s, i) => {
            const nextX = steps[i + 1].x;
            const isCurrent = i === phase;

            let edgeColor: string = inactive.edge;
            if (i < phase) {
              edgeColor =
                i === 1 ? state.running : i === 2 ? state.success : brand.blue;
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

            let nodeFill: string = fill.inactiveNode;
            let stroke: string = inactive.edge;
            let textColor: string = inactive.text;
            let subColor: string = inactive.stroke;

            if (isActive) {
              nodeFill = phaseColors[i].fill;
              stroke = phaseColors[i].stroke;
              textColor = phaseColors[i].text;
              subColor = phaseColors[i].sub;
            } else if (isPast) {
              nodeFill = phaseColors[i].fill;
              stroke = phaseColors[i].stroke;
              textColor = phaseColors[i].text;
              subColor = phaseColors[i].sub;
            }

            return (
              <g key={`node-${i}`}>
                <rect
                  x={s.x}
                  y={y}
                  width={nodeW}
                  height={nodeH}
                  rx={rx}
                  fill={nodeFill}
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

                {/* Crash indicator (SVG bolt) */}
                {i === 2 && isActive && (
                  <g transform={`translate(${s.x + nodeW / 2 - 7}, ${y - 22})`}>
                    <svg
                      width="14"
                      height="14"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke={state.failed}
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
                    </svg>
                  </g>
                )}

                {/* Checkpoint indicator (SVG save/disk) */}
                {i === 1 && (isActive || isPast) && (
                  <g transform={`translate(${s.x + nodeW / 2 - 7}, ${y - 20})`}>
                    <svg
                      width="14"
                      height="14"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke={state.runningLight}
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
                      <polyline points="17 21 17 13 7 13 7 21" />
                      <polyline points="7 3 7 8 15 8" />
                    </svg>
                  </g>
                )}

                {/* Timeline dot */}
                <circle
                  cx={s.x + nodeW / 2}
                  cy={y + nodeH + 20}
                  r={i <= phase ? 4 : 3}
                  fill={
                    i < phase
                      ? i === 2
                        ? state.failed
                        : state.success
                      : i === phase
                        ? phaseColors[i].stroke
                        : inactive.dot
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
                stroke={state.success}
                strokeWidth="1.5"
                strokeDasharray="4 3"
              />
              <text
                x={(steps[1].x + steps[3].x) / 2 + nodeW / 2}
                y={y + nodeH + 62}
                textAnchor="middle"
                fill={state.successLight}
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
