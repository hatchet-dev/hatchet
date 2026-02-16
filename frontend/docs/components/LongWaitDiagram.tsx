import React, { useState, useEffect } from "react";

type WaitType = "sleep" | "event";

const LongWaitDiagram: React.FC = () => {
  const [waitType, setWaitType] = useState<WaitType>("sleep");
  const [phase, setPhase] = useState(0); // 0=running, 1=waiting, 2=resumed, 3=complete

  useEffect(() => {
    const durations = [1200, 2400, 1200, 1200];
    const timer = setTimeout(() => {
      setPhase((prev) => (prev + 1) % 4);
    }, durations[phase]);
    return () => clearTimeout(timer);
  }, [phase]);

  const nodeW = 130;
  const nodeH = 50;
  const rx = 10;

  // Positions
  const taskX = 30;
  const taskY = 80;
  const waitX = 220;
  const waitY = 80;
  const resumeX = 440;
  const resumeY = 80;
  const completeX = 600;
  const completeY = 80;

  const isWaiting = phase === 1;
  const isResumed = phase === 2;
  const isDone = phase === 3;

  const waitLabel = waitType === "sleep" ? "Sleep 24h" : "Wait for Event";
  const waitSublabel =
    waitType === "sleep" ? "durable pause" : "external signal";
  const triggerLabel =
    waitType === "sleep" ? "time elapsed" : "event received";

  return (
    <div className="my-8 flex flex-col items-center gap-4">
      {/* Toggle */}
      <div className="flex items-center gap-2 rounded-lg border border-neutral-700/40 bg-neutral-900/50 px-4 py-2">
        <span className="text-xs font-medium text-gray-400">Wait type:</span>
        {(["sleep", "event"] as WaitType[]).map((type) => (
          <button
            key={type}
            onClick={() => {
              setWaitType(type);
              setPhase(0);
            }}
            className="rounded-md px-3 py-1 text-xs font-medium transition-all duration-200"
            style={{
              backgroundColor:
                waitType === type
                  ? type === "sleep"
                    ? "rgba(99,102,241,0.3)"
                    : "rgba(245,158,11,0.3)"
                  : "rgba(50,50,50,0.3)",
              border: `1px solid ${
                waitType === type
                  ? type === "sleep"
                    ? "rgb(99,102,241)"
                    : "rgb(245,158,11)"
                  : "#555"
              }`,
              color:
                waitType === type
                  ? type === "sleep"
                    ? "#c7d2fe"
                    : "#fcd34d"
                  : "#666",
            }}
          >
            {type === "sleep" ? "Durable Sleep" : "Durable Event"}
          </button>
        ))}
        <span
          className="ml-2 text-xs transition-colors duration-300"
          style={{
            color: isDone
              ? "#6ee7b7"
              : isWaiting
                ? waitType === "sleep"
                  ? "#c7d2fe"
                  : "#fcd34d"
                : "#6b7280",
          }}
        >
          {isDone
            ? "complete!"
            : isResumed
              ? "resuming..."
              : isWaiting
                ? "waiting..."
                : "running..."}
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
            <linearGradient
              id="lw-indigo"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop
                offset="0%"
                stopColor="rgb(99,102,241)"
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor="rgb(129,140,248)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient
              id="lw-amber"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop
                offset="0%"
                stopColor="rgb(245,158,11)"
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor="rgb(252,211,77)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient
              id="lw-emerald"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop
                offset="0%"
                stopColor="rgb(16,185,129)"
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor="rgb(52,211,153)"
                stopOpacity="0.3"
              />
            </linearGradient>
          </defs>

          <style>{`
            @keyframes lw-dash {
              to { stroke-dashoffset: -20; }
            }
            .lw-flow {
              stroke-dasharray: 8 6;
              animation: lw-dash 0.8s linear infinite;
            }
            @keyframes lw-pulse {
              0%, 100% { opacity: 0.3; }
              50% { opacity: 1; }
            }
            .lw-pulse {
              animation: lw-pulse 1.5s ease-in-out infinite;
            }
          `}</style>

          {/* Task box */}
          <rect
            x={taskX}
            y={taskY}
            width={nodeW}
            height={nodeH}
            rx={rx}
            fill={
              phase === 0 ? "rgba(49,46,129,0.3)" : "rgba(49,46,129,0.15)"
            }
            stroke={phase === 0 ? "url(#lw-indigo)" : "#555"}
            strokeWidth="1.5"
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={taskX + nodeW / 2}
            y={taskY + nodeH / 2 - 2}
            textAnchor="middle"
            fill={phase === 0 ? "#c7d2fe" : "#888"}
            fontSize="13"
            fontWeight="500"
          >
            Task Runs
          </text>
          <text
            x={taskX + nodeW / 2}
            y={taskY + nodeH / 2 + 14}
            textAnchor="middle"
            fill={phase === 0 ? "#818cf8" : "#666"}
            fontSize="9"
          >
            do work
          </text>

          {/* Edge: Task -> Wait */}
          <path
            d={`M ${taskX + nodeW + 2} ${taskY + nodeH / 2} L ${waitX - 2} ${waitY + nodeH / 2}`}
            fill="none"
            stroke={phase === 0 ? "rgb(129,140,248)" : "#555"}
            strokeWidth="2"
            className={phase === 0 ? "lw-flow" : ""}
            style={{ transition: "stroke 0.4s ease" }}
          />

          {/* Wait box - larger, distinctive */}
          <rect
            x={waitX}
            y={waitY - 15}
            width={nodeW + 60}
            height={nodeH + 30}
            rx={rx}
            fill={
              isWaiting
                ? waitType === "sleep"
                  ? "rgba(49,46,129,0.25)"
                  : "rgba(120,53,15,0.25)"
                : "rgba(30,30,30,0.15)"
            }
            stroke={
              isWaiting
                ? waitType === "sleep"
                  ? "url(#lw-indigo)"
                  : "url(#lw-amber)"
                : "#555"
            }
            strokeWidth="1.5"
            strokeDasharray={isWaiting ? "6 4" : "none"}
            style={{ transition: "all 0.4s ease" }}
          />
          {/* Pause icon */}
          {isWaiting && (
            <g className="lw-pulse">
              <rect
                x={waitX + (nodeW + 60) / 2 - 12}
                y={waitY + 28}
                width="5"
                height="16"
                rx="1"
                fill={waitType === "sleep" ? "#818cf8" : "#fcd34d"}
              />
              <rect
                x={waitX + (nodeW + 60) / 2 + 4}
                y={waitY + 28}
                width="5"
                height="16"
                rx="1"
                fill={waitType === "sleep" ? "#818cf8" : "#fcd34d"}
              />
            </g>
          )}
          <text
            x={waitX + (nodeW + 60) / 2}
            y={waitY + 10}
            textAnchor="middle"
            fill={
              isWaiting
                ? waitType === "sleep"
                  ? "#c7d2fe"
                  : "#fcd34d"
                : "#888"
            }
            fontSize="13"
            fontWeight="500"
            style={{ transition: "fill 0.4s ease" }}
          >
            {waitLabel}
          </text>
          {!isWaiting && (
            <text
              x={waitX + (nodeW + 60) / 2}
              y={waitY + 28}
              textAnchor="middle"
              fill="#666"
              fontSize="9"
            >
              {waitSublabel}
            </text>
          )}

          {/* Trigger label below wait box */}
          <text
            x={waitX + (nodeW + 60) / 2}
            y={waitY + nodeH + 30}
            textAnchor="middle"
            fill={isResumed || isDone ? "#6ee7b7" : "#555"}
            fontSize="9"
            style={{ transition: "fill 0.4s ease" }}
          >
            {isResumed || isDone ? triggerLabel : ""}
          </text>

          {/* Edge: Wait -> Resume */}
          <path
            d={`M ${waitX + nodeW + 62} ${waitY + nodeH / 2} L ${resumeX - 2} ${resumeY + nodeH / 2}`}
            fill="none"
            stroke={isResumed ? "rgb(16,185,129)" : "#444"}
            strokeWidth="2"
            className={isResumed ? "lw-flow" : ""}
            style={{
              opacity: isResumed || isDone ? 1 : 0.2,
              transition: "all 0.4s ease",
            }}
          />

          {/* Resume box */}
          <rect
            x={resumeX}
            y={resumeY}
            width={nodeW}
            height={nodeH}
            rx={rx}
            fill={
              isResumed ? "rgba(6,78,59,0.3)" : "rgba(30,30,30,0.15)"
            }
            stroke={isResumed ? "url(#lw-emerald)" : "#444"}
            strokeWidth="1.5"
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={resumeX + nodeW / 2}
            y={resumeY + nodeH / 2 - 2}
            textAnchor="middle"
            fill={isResumed ? "#a7f3d0" : "#666"}
            fontSize="13"
            fontWeight="500"
            style={{ transition: "fill 0.4s ease" }}
          >
            Resume
          </text>
          <text
            x={resumeX + nodeW / 2}
            y={resumeY + nodeH / 2 + 14}
            textAnchor="middle"
            fill={isResumed ? "#6ee7b7" : "#555"}
            fontSize="9"
            style={{ transition: "fill 0.4s ease" }}
          >
            continue work
          </text>

          {/* Edge: Resume -> Complete */}
          <path
            d={`M ${resumeX + nodeW + 2} ${resumeY + nodeH / 2} L ${completeX - 2} ${completeY + nodeH / 2}`}
            fill="none"
            stroke={isDone ? "rgb(16,185,129)" : "#444"}
            strokeWidth="2"
            className={isDone ? "lw-flow" : ""}
            style={{
              opacity: isDone ? 1 : 0.2,
              transition: "all 0.4s ease",
            }}
          />

          {/* Complete box */}
          <rect
            x={completeX}
            y={completeY}
            width={nodeW}
            height={nodeH}
            rx={rx}
            fill={isDone ? "rgba(6,78,59,0.3)" : "rgba(30,30,30,0.15)"}
            stroke={isDone ? "url(#lw-emerald)" : "#444"}
            strokeWidth="1.5"
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={completeX + nodeW / 2}
            y={completeY + nodeH / 2 - 2}
            textAnchor="middle"
            fill={isDone ? "#a7f3d0" : "#666"}
            fontSize="13"
            fontWeight="500"
            style={{ transition: "fill 0.4s ease" }}
          >
            Complete
          </text>
          <text
            x={completeX + nodeW / 2}
            y={completeY + nodeH / 2 + 14}
            textAnchor="middle"
            fill={isDone ? "#6ee7b7" : "#555"}
            fontSize="9"
            style={{ transition: "fill 0.4s ease" }}
          >
            return result
          </text>
        </svg>
      </div>
    </div>
  );
};

export default LongWaitDiagram;
