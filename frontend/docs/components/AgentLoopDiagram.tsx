import React, { useState, useEffect } from "react";

const PHASES = ["reason", "act", "observe", "decide"] as const;
type Phase = (typeof PHASES)[number];

const PHASE_CONFIG: Record<
  Phase,
  { label: string; color: string; icon: string }
> = {
  reason: { label: "Reason", color: "#818cf8", icon: "🧠" },
  act: { label: "Act", color: "#38bdf8", icon: "⚡" },
  observe: { label: "Observe", color: "#fbbf24", icon: "👁" },
  decide: { label: "Decide", color: "#34d399", icon: "🔀" },
};

const AgentLoopDiagram: React.FC = () => {
  const [phase, setPhase] = useState<Phase>("reason");
  const [iteration, setIteration] = useState(1);

  useEffect(() => {
    const interval = setInterval(() => {
      setPhase((prev) => {
        const idx = PHASES.indexOf(prev);
        if (idx === PHASES.length - 1) {
          setIteration((i) => (i >= 3 ? 1 : i + 1));
          return PHASES[0];
        }
        return PHASES[idx + 1];
      });
    }, 1200);
    return () => clearInterval(interval);
  }, []);

  const cx = 200;
  const cy = 140;
  const r = 90;

  const nodePositions = PHASES.map((_, i) => {
    const angle = (i * Math.PI * 2) / PHASES.length - Math.PI / 2;
    return { x: cx + r * Math.cos(angle), y: cy + r * Math.sin(angle) };
  });

  return (
    <div
      className="my-6 rounded-xl border p-6"
      style={{
        borderColor: "rgba(99,102,241,0.2)",
        backgroundColor: "rgba(49,46,129,0.04)",
      }}
    >
      <svg
        viewBox="0 0 400 280"
        className="mx-auto w-full"
        style={{ maxWidth: 500 }}
      >
        {/* Circular arrows between phases */}
        {PHASES.map((_, i) => {
          const from = nodePositions[i];
          const to = nodePositions[(i + 1) % PHASES.length];
          const midX = (from.x + to.x) / 2;
          const midY = (from.y + to.y) / 2;
          const dx = to.x - from.x;
          const dy = to.y - from.y;
          const len = Math.sqrt(dx * dx + dy * dy);
          const nx = -dy / len;
          const ny = dx / len;
          const bulge = 20;
          const cpX = midX + nx * bulge;
          const cpY = midY + ny * bulge;

          const phaseIdx = PHASES.indexOf(phase);
          const isActive = i === phaseIdx;

          return (
            <path
              key={`arrow-${i}`}
              d={`M ${from.x} ${from.y} Q ${cpX} ${cpY} ${to.x} ${to.y}`}
              fill="none"
              stroke={isActive ? PHASE_CONFIG[PHASES[i]].color : "#374151"}
              strokeWidth={isActive ? 2.5 : 1.5}
              strokeDasharray={isActive ? "6 3" : "none"}
              opacity={isActive ? 1 : 0.4}
              style={{
                transition: "all 0.4s ease",
              }}
            />
          );
        })}

        {/* Phase nodes */}
        {PHASES.map((p, i) => {
          const pos = nodePositions[i];
          const config = PHASE_CONFIG[p];
          const isActive = phase === p;

          return (
            <g key={p}>
              {/* Glow */}
              {isActive && (
                <circle
                  cx={pos.x}
                  cy={pos.y}
                  r={36}
                  fill={config.color}
                  opacity={0.15}
                >
                  <animate
                    attributeName="r"
                    values="36;42;36"
                    dur="1.5s"
                    repeatCount="indefinite"
                  />
                  <animate
                    attributeName="opacity"
                    values="0.15;0.08;0.15"
                    dur="1.5s"
                    repeatCount="indefinite"
                  />
                </circle>
              )}
              {/* Circle */}
              <circle
                cx={pos.x}
                cy={pos.y}
                r={30}
                fill={isActive ? `${config.color}22` : "#1f2937"}
                stroke={isActive ? config.color : "#374151"}
                strokeWidth={isActive ? 2 : 1.5}
                style={{ transition: "all 0.4s ease" }}
              />
              {/* Icon */}
              <text
                x={pos.x}
                y={pos.y - 4}
                textAnchor="middle"
                fontSize="16"
                style={{ transition: "all 0.4s ease" }}
              >
                {config.icon}
              </text>
              {/* Label */}
              <text
                x={pos.x}
                y={pos.y + 16}
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

        {/* Center iteration counter */}
        <text
          x={cx}
          y={cy - 8}
          textAnchor="middle"
          fontSize="11"
          fill="#6b7280"
        >
          Iteration
        </text>
        <text
          x={cx}
          y={cy + 10}
          textAnchor="middle"
          fontSize="18"
          fontWeight={700}
          fill="#e5e7eb"
        >
          {iteration}/3
        </text>
      </svg>

      {/* Status bar */}
      <div className="mt-4 flex items-center justify-center gap-3">
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
            <span>{PHASE_CONFIG[p].icon}</span>
            {PHASE_CONFIG[p].label}
          </div>
        ))}
      </div>
    </div>
  );
};

export default AgentLoopDiagram;
