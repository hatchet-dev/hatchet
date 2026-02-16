import React, { useState, useEffect } from "react";

const SOURCES = [
  { label: "Webhook", icon: "🔗", color: "#818cf8" },
  { label: "Cron", icon: "⏰", color: "#38bdf8" },
  { label: "Event", icon: "📡", color: "#fbbf24" },
];

const EventDrivenDiagram: React.FC = () => {
  const [activeSource, setActiveSource] = useState(0);
  const [pulseVisible, setPulseVisible] = useState(false);

  useEffect(() => {
    const interval = setInterval(() => {
      setPulseVisible(true);
      setTimeout(() => setPulseVisible(false), 800);
      setTimeout(() => {
        setActiveSource((prev) => (prev + 1) % SOURCES.length);
      }, 1000);
    }, 2000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div
      className="my-6 rounded-xl border p-6"
      style={{
        borderColor: "rgba(99,102,241,0.2)",
        backgroundColor: "rgba(49,46,129,0.04)",
      }}
    >
      <svg
        viewBox="0 0 440 180"
        className="mx-auto w-full"
        style={{ maxWidth: 520 }}
      >
        {/* Event sources */}
        {SOURCES.map((src, i) => {
          const y = 30 + i * 55;
          const isActive = i === activeSource;

          return (
            <g key={src.label}>
              {/* Source box */}
              <rect
                x={20}
                y={y}
                width={90}
                height={40}
                rx={8}
                fill={isActive ? `${src.color}15` : "#1f2937"}
                stroke={isActive ? src.color : "#374151"}
                strokeWidth={isActive ? 2 : 1}
                style={{ transition: "all 0.4s ease" }}
              />
              <text x={45} y={y + 18} fontSize="14" textAnchor="middle">
                {src.icon}
              </text>
              <text
                x={75}
                y={y + 27}
                fontSize="10"
                fill={isActive ? src.color : "#9ca3af"}
                fontWeight={isActive ? 600 : 400}
                style={{ transition: "all 0.4s ease" }}
              >
                {src.label}
              </text>

              {/* Arrow to Hatchet */}
              <line
                x1={112}
                y1={y + 20}
                x2={170}
                y2={90}
                stroke={isActive ? src.color : "#374151"}
                strokeWidth={isActive ? 2 : 1}
                opacity={isActive ? 1 : 0.3}
                style={{ transition: "all 0.4s ease" }}
              />

              {/* Pulse dot */}
              {isActive && pulseVisible && (
                <circle r="4" fill={src.color}>
                  <animateMotion
                    dur="0.8s"
                    repeatCount="1"
                    path={`M 112 ${y + 20} L 170 90`}
                  />
                  <animate
                    attributeName="opacity"
                    values="1;0"
                    dur="0.8s"
                    repeatCount="1"
                  />
                </circle>
              )}
            </g>
          );
        })}

        {/* Hatchet engine center */}
        <rect
          x={170}
          y={65}
          width={100}
          height={50}
          rx={12}
          fill="rgba(99,102,241,0.1)"
          stroke="#818cf8"
          strokeWidth={2}
        />
        <text x={220} y={85} textAnchor="middle" fontSize="10" fill="#c7d2fe">
          Hatchet
        </text>
        <text x={220} y={100} textAnchor="middle" fontSize="10" fill="#818cf8" fontWeight={600}>
          Engine
        </text>

        {/* Workers */}
        {[0, 1, 2].map((i) => {
          const y = 30 + i * 55;
          const isActive = pulseVisible && i === activeSource;

          return (
            <g key={`worker-${i}`}>
              {/* Arrow from Hatchet to worker */}
              <line
                x1={272}
                y1={90}
                x2={330}
                y2={y + 20}
                stroke={isActive ? "#34d399" : "#374151"}
                strokeWidth={isActive ? 2 : 1}
                opacity={isActive ? 1 : 0.3}
                style={{ transition: "all 0.4s ease" }}
              />

              {/* Worker box */}
              <rect
                x={330}
                y={y}
                width={90}
                height={40}
                rx={8}
                fill={isActive ? "rgba(52,211,153,0.1)" : "#1f2937"}
                stroke={isActive ? "#34d399" : "#374151"}
                strokeWidth={isActive ? 2 : 1}
                style={{ transition: "all 0.4s ease" }}
              />
              <text
                x={375}
                y={y + 18}
                textAnchor="middle"
                fontSize="14"
              >
                ⚙️
              </text>
              <text
                x={375}
                y={y + 33}
                textAnchor="middle"
                fontSize="9"
                fill={isActive ? "#34d399" : "#9ca3af"}
                style={{ transition: "all 0.4s ease" }}
              >
                Worker {i + 1}
              </text>
            </g>
          );
        })}
      </svg>

      {/* Legend */}
      <div className="mt-3 flex items-center justify-center gap-4">
        {SOURCES.map((src, i) => (
          <div
            key={src.label}
            className="flex items-center gap-1.5 text-xs"
            style={{
              color: i === activeSource ? src.color : "#6b7280",
              transition: "color 0.4s ease",
            }}
          >
            <span>{src.icon}</span>
            {src.label}
          </div>
        ))}
      </div>
    </div>
  );
};

export default EventDrivenDiagram;
