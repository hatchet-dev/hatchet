import React, { useState, useEffect } from "react";

const PHASES = ["thought", "action", "observation"] as const;
type Phase = (typeof PHASES)[number];

const PHASE_CONFIG: Record<Phase, { label: string; color: string }> = {
  thought: { label: "Thought", color: "#818cf8" },
  action: { label: "Action", color: "#38bdf8" },
  observation: { label: "Observation", color: "#fbbf24" },
};

/** Small SVG icons rendered inline, no emojis */
const PhaseIcon: React.FC<{ phase: Phase; active: boolean }> = ({
  phase,
  active,
}) => {
  const color = active ? PHASE_CONFIG[phase].color : "#6b7280";
  const size = 18;

  switch (phase) {
    case "thought":
      // Lightbulb icon
      return (
        <svg
          width={size}
          height={size}
          viewBox="0 0 24 24"
          fill="none"
          stroke={color}
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M9 18h6" />
          <path d="M10 22h4" />
          <path d="M12 2a7 7 0 0 0-4 12.7V17h8v-2.3A7 7 0 0 0 12 2z" />
        </svg>
      );
    case "action":
      // Zap/bolt icon
      return (
        <svg
          width={size}
          height={size}
          viewBox="0 0 24 24"
          fill="none"
          stroke={color}
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
        </svg>
      );
    case "observation":
      // Eye icon
      return (
        <svg
          width={size}
          height={size}
          viewBox="0 0 24 24"
          fill="none"
          stroke={color}
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
          <circle cx="12" cy="12" r="3" />
        </svg>
      );
  }
};

const AgentLoopDiagram: React.FC = () => {
  const [phaseIdx, setPhaseIdx] = useState(0);
  const [iteration, setIteration] = useState(1);

  useEffect(() => {
    const interval = setInterval(() => {
      setPhaseIdx((prev) => {
        if (prev === PHASES.length - 1) {
          setIteration((i) => (i >= 3 ? 1 : i + 1));
          return 0;
        }
        return prev + 1;
      });
    }, 1400);
    return () => clearInterval(interval);
  }, []);

  const phase = PHASES[phaseIdx];

  // Horizontal layout: 3 nodes evenly spaced
  const svgW = 520;
  const svgH = 160;
  const nodeY = 70;
  const nodeSpacing = 160;
  const startX = 100;

  const nodes = PHASES.map((_, i) => ({
    x: startX + i * nodeSpacing,
    y: nodeY,
  }));

  return (
    <div
      className="my-6 rounded-xl border p-6"
      style={{
        borderColor: "rgba(99,102,241,0.2)",
        backgroundColor: "rgba(49,46,129,0.04)",
      }}
    >
      <svg
        viewBox={`0 0 ${svgW} ${svgH}`}
        className="mx-auto w-full"
        style={{ maxWidth: 560 }}
      >
        <defs>
          <marker
            id="arrow"
            viewBox="0 0 10 7"
            refX="9"
            refY="3.5"
            markerWidth="8"
            markerHeight="6"
            orient="auto-start-reverse"
          >
            <polygon points="0 0, 10 3.5, 0 7" fill="#4b5563" />
          </marker>
          <marker
            id="arrow-active"
            viewBox="0 0 10 7"
            refX="9"
            refY="3.5"
            markerWidth="8"
            markerHeight="6"
            orient="auto-start-reverse"
          >
            <polygon points="0 0, 10 3.5, 0 7" fill="#818cf8" />
          </marker>
        </defs>

        {/* Forward arrows between nodes */}
        {nodes.slice(0, -1).map((from, i) => {
          const to = nodes[i + 1];
          const isActive = phaseIdx === i;
          return (
            <line
              key={`fwd-${i}`}
              x1={from.x + 34}
              y1={from.y}
              x2={to.x - 34}
              y2={to.y}
              stroke={isActive ? PHASE_CONFIG[PHASES[i]].color : "#4b5563"}
              strokeWidth={isActive ? 2 : 1.5}
              markerEnd={isActive ? "url(#arrow-active)" : "url(#arrow)"}
              opacity={isActive ? 1 : 0.5}
              style={{ transition: "all 0.4s ease" }}
            />
          );
        })}

        {/* Return arrow: curved path from Observation back to Thought */}
        {(() => {
          const from = nodes[nodes.length - 1];
          const to = nodes[0];
          const isActive = phaseIdx === PHASES.length - 1;
          const curveY = nodeY + 58;
          return (
            <path
              d={`M ${from.x} ${from.y + 30} C ${from.x} ${curveY + 10}, ${to.x} ${curveY + 10}, ${to.x} ${to.y + 30}`}
              fill="none"
              stroke={isActive ? PHASE_CONFIG.observation.color : "#4b5563"}
              strokeWidth={isActive ? 2 : 1.5}
              strokeDasharray="6 4"
              opacity={isActive ? 1 : 0.35}
              style={{ transition: "all 0.4s ease" }}
            />
          );
        })()}

        {/* Loop label on return arrow */}
        <text
          x={svgW / 2}
          y={nodeY + 78}
          textAnchor="middle"
          fontSize="10"
          fill="#6b7280"
          fontStyle="italic"
        >
          iteration {iteration}/3
        </text>

        {/* Phase nodes */}
        {PHASES.map((p, i) => {
          const pos = nodes[i];
          const config = PHASE_CONFIG[p];
          const isActive = phase === p;

          return (
            <g key={p}>
              {/* Glow ring */}
              {isActive && (
                <rect
                  x={pos.x - 36}
                  y={pos.y - 36}
                  width={72}
                  height={72}
                  rx={16}
                  fill={config.color}
                  opacity={0.1}
                >
                  <animate
                    attributeName="opacity"
                    values="0.1;0.05;0.1"
                    dur="1.8s"
                    repeatCount="indefinite"
                  />
                </rect>
              )}
              {/* Node box */}
              <rect
                x={pos.x - 30}
                y={pos.y - 30}
                width={60}
                height={60}
                rx={14}
                fill={isActive ? `${config.color}18` : "#1f2937"}
                stroke={isActive ? config.color : "#374151"}
                strokeWidth={isActive ? 2 : 1.5}
                style={{ transition: "all 0.4s ease" }}
              />
              {/* Icon */}
              <foreignObject
                x={pos.x - 9}
                y={pos.y - 18}
                width={18}
                height={18}
              >
                <PhaseIcon phase={p} active={isActive} />
              </foreignObject>
              {/* Label */}
              <text
                x={pos.x}
                y={pos.y + 14}
                textAnchor="middle"
                fontSize="10"
                fontWeight={isActive ? 600 : 400}
                fill={isActive ? config.color : "#9ca3af"}
                style={{ transition: "all 0.4s ease" }}
              >
                {config.label}
              </text>
            </g>
          );
        })}
      </svg>

      {/* Status indicators */}
      <div className="mt-3 flex items-center justify-center gap-3">
        {PHASES.map((p) => (
          <div
            key={p}
            className="flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium"
            style={{
              backgroundColor:
                phase === p
                  ? `${PHASE_CONFIG[p].color}20`
                  : "rgba(55,65,81,0.3)",
              color: phase === p ? PHASE_CONFIG[p].color : "#6b7280",
              border: `1px solid ${phase === p ? `${PHASE_CONFIG[p].color}40` : "transparent"}`,
              transition: "all 0.4s ease",
            }}
          >
            <PhaseIcon phase={p} active={phase === p} />
            {PHASE_CONFIG[p].label}
          </div>
        ))}
      </div>
    </div>
  );
};

export default AgentLoopDiagram;
