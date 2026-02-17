import React, { useState, useEffect } from "react";

type SourceType = "Webhook" | "Cron" | "Event";

const SOURCES: { label: SourceType; color: string }[] = [
  { label: "Webhook", color: "#818cf8" },
  { label: "Cron", color: "#38bdf8" },
  { label: "Event", color: "#fbbf24" },
];

const SourceIcon: React.FC<{
  type: SourceType;
  color: string;
  size?: number;
}> = ({ type, color, size = 14 }) => {
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

  switch (type) {
    case "Webhook":
      // Link icon
      return (
        <svg {...props}>
          <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
          <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
        </svg>
      );
    case "Cron":
      // Clock icon
      return (
        <svg {...props}>
          <circle cx="12" cy="12" r="10" />
          <polyline points="12 6 12 12 16 14" />
        </svg>
      );
    case "Event":
      // Signal/broadcast icon
      return (
        <svg {...props}>
          <path d="M5.5 16.5a2.12 2.12 0 0 1 0-3" />
          <path d="M18.5 16.5a2.12 2.12 0 0 0 0-3" />
          <path d="M3 19a4.24 4.24 0 0 1 0-8" />
          <path d="M21 19a4.24 4.24 0 0 0 0-8" />
          <circle cx="12" cy="15" r="2" />
          <line x1="12" y1="13" x2="12" y2="6" />
        </svg>
      );
  }
};

const GearIcon: React.FC<{ color: string; size?: number }> = ({
  color,
  size = 14,
}) => (
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
    <circle cx="12" cy="12" r="3" />
    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
  </svg>
);

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
              {/* Icon via foreignObject */}
              <foreignObject x={30} y={y + 10} width={14} height={14}>
                <SourceIcon
                  type={src.label}
                  color={isActive ? src.color : "#6b7280"}
                />
              </foreignObject>
              <text
                x={72}
                y={y + 25}
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
        <text
          x={220}
          y={100}
          textAnchor="middle"
          fontSize="10"
          fill="#818cf8"
          fontWeight={600}
        >
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
              {/* Gear icon */}
              <foreignObject x={368} y={y + 5} width={14} height={14}>
                <GearIcon color={isActive ? "#34d399" : "#6b7280"} />
              </foreignObject>
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
            <SourceIcon
              type={src.label}
              color={i === activeSource ? src.color : "#6b7280"}
              size={12}
            />
            {src.label}
          </div>
        ))}
      </div>
    </div>
  );
};

export default EventDrivenDiagram;
