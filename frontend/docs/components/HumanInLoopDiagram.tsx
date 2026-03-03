import React, { useState, useEffect } from "react";
import { brand, state, fill, inactive, gradient } from "./diagram-colors";

const HumanInLoopDiagram: React.FC = () => {
  const [phase, setPhase] = useState(0); // 0=agent, 1=waiting, 2=human, 3=resume, 4=complete

  useEffect(() => {
    const durations = [1200, 2200, 1000, 1200, 1200];
    const timer = setTimeout(() => {
      setPhase((prev) => (prev + 1) % 5);
    }, durations[phase]);
    return () => clearTimeout(timer);
  }, [phase]);

  const nodeW = 88;
  const nodeH = 40;
  const rx = 10;
  const gap = 28;

  const paddingX = 32;
  const agentX = paddingX;
  const waitX = agentX + nodeW + gap;
  const waitW = 140;
  const resumeX = waitX + waitW + gap;
  const completeX = resumeX + nodeW + gap;

  const flowY = 115;
  const humanH = 48;
  const humanW = 82;
  const humanY = 16;
  const humanX = waitX + waitW / 2 - humanW / 2;
  const totalW = completeX + nodeW + paddingX;

  const isAgent = phase === 0;
  const isWaiting = phase === 1;
  const isHuman = phase === 2;
  const isResumed = phase === 3;
  const isDone = phase === 4;

  const waitActive = isWaiting || isHuman;

  return (
    <div className="my-8 flex flex-col items-center gap-4">
      <div
        className="text-xs font-medium transition-colors duration-300"
        style={{
          color: isDone
            ? "#4ADE80"
            : isHuman
              ? "#FACC15"
              : isWaiting
                ? "#EAB308"
                : "#64748B",
        }}
      >
        {isDone
          ? "complete"
          : isResumed
            ? "resuming..."
            : isHuman
              ? "human approves"
              : isWaiting
                ? "slot freed — waiting"
                : "agent proposes"}
      </div>

      <div className="w-full max-w-3xl rounded-xl border border-neutral-700/40 bg-neutral-900/30 p-6 backdrop-blur-sm flex justify-center">
        <svg
          viewBox={`0 0 ${totalW} 170`}
          className="max-w-full h-auto"
          preserveAspectRatio="xMidYMid meet"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient
              id="hitl-blue"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop offset="0%" stopColor="rgb(51, 146, 255)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(133, 189, 255)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="hitl-yellow" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="rgb(234, 179, 8)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(250, 204, 21)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient
              id="hitl-green"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop offset="0%" stopColor="rgb(34, 197, 94)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(74, 222, 128)"
                stopOpacity="0.3"
              />
            </linearGradient>
          </defs>

          <style>{`
            @keyframes hitl-dash {
              to { stroke-dashoffset: -20; }
            }
            .hitl-flow {
              stroke-dasharray: 8 6;
              animation: hitl-dash 0.8s linear infinite;
            }
            @keyframes hitl-pulse {
              0%, 100% { opacity: 0.4; }
              50% { opacity: 1; }
            }
            .hitl-pulse {
              animation: hitl-pulse 1.2s ease-in-out infinite;
            }
          `}</style>

          {/* Agent box */}
          <rect
            x={agentX}
            y={flowY - nodeH / 2}
            width={nodeW}
            height={nodeH}
            rx={rx}
            fill={isAgent ? "rgba(10, 16, 41, 0.3)" : "rgba(10, 16, 41, 0.15)"}
            stroke={isAgent ? "url(#hitl-blue)" : "#1C2B4A"}
            strokeWidth="1.5"
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={agentX + nodeW / 2}
            y={flowY - 4}
            textAnchor="middle"
            fill={isAgent ? "#B8D9FF" : "#4A6080"}
            fontSize="12"
            fontWeight="500"
          >
            Agent
          </text>
          <text
            x={agentX + nodeW / 2}
            y={flowY + 12}
            textAnchor="middle"
            fill={isAgent ? "#3392FF" : "#64748B"}
            fontSize="9"
          >
            proposes
          </text>

          {/* Edge: Agent -> Wait */}
          <path
            d={`M ${agentX + nodeW + 2} ${flowY} L ${waitX - 2} ${flowY}`}
            fill="none"
            stroke={isAgent ? "rgb(51, 146, 255)" : "#1C2B4A"}
            strokeWidth="2"
            className={isAgent ? "hitl-flow" : ""}
            style={{ transition: "stroke 0.4s ease" }}
          />

          {/* Wait for Approval box (paused block) */}
          <rect
            x={waitX}
            y={flowY - nodeH / 2 - 12}
            width={waitW}
            height={nodeH + 24}
            rx={rx}
            fill={waitActive ? "rgba(234, 179, 8, 0.25)" : "rgba(10, 16, 41, 0.15)"}
            stroke={waitActive ? "url(#hitl-yellow)" : "#1C2B4A"}
            strokeWidth="1.5"
            strokeDasharray={waitActive ? "6 4" : "none"}
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={waitX + waitW / 2}
            y={flowY - 10}
            textAnchor="middle"
            fill={waitActive ? "#FACC15" : "#4A6080"}
            fontSize="11"
            fontWeight="500"
          >
            Wait for Approval
          </text>
          <text
            x={waitX + waitW / 2}
            y={flowY + 6}
            textAnchor="middle"
            fill={waitActive ? "#EAB308" : "#64748B"}
            fontSize="9"
          >
            {isWaiting ? "slot freed" : "WaitForEvent"}
          </text>
          {isWaiting && (
            <g className="hitl-pulse">
              <rect
                x={waitX + waitW / 2 - 10}
                y={flowY + 10}
                width="5"
                height="12"
                rx="1"
                fill="#FACC15"
              />
              <rect
                x={waitX + waitW / 2 + 2}
                y={flowY + 10}
                width="5"
                height="12"
                rx="1"
                fill="#FACC15"
              />
            </g>
          )}

          {/* Human above the paused block */}
          <g
            style={{
              opacity: waitActive ? 1 : 0.5,
              transition: "opacity 0.4s ease",
            }}
          >
            {/* Dashed line: Human bottom -> top of Wait block */}
            <line
              x1={waitX + waitW / 2}
              y1={flowY - nodeH / 2 - 12}
              x2={waitX + waitW / 2}
              y2={humanY + humanH}
              stroke={isHuman ? "#FACC15" : "#64748B"}
              strokeWidth="1.5"
              strokeDasharray="4 4"
              opacity={waitActive ? 0.8 : 0.4}
            />
            {/* Human box */}
            <rect
              x={humanX}
              y={humanY}
              width={humanW}
              height={humanH}
              rx={rx}
              fill={isHuman ? "rgba(234, 179, 8, 0.25)" : "rgba(10, 16, 41, 0.2)"}
              stroke={isHuman ? "url(#hitl-yellow)" : "#1C2B4A"}
              strokeWidth="1.5"
              style={{ transition: "all 0.4s ease" }}
            />
            {/* Person icon */}
            <g
              transform={`translate(${humanX + humanW / 2 - 8}, ${humanY + 6})`}
            >
              <circle
                cx="8"
                cy="5"
                r="4"
                fill="none"
                stroke={isHuman ? "#FACC15" : "#64748B"}
                strokeWidth="1.5"
              />
              <path
                d="M2 18c0-3 2-6 6-6s6 3 6 6"
                fill="none"
                stroke={isHuman ? "#FACC15" : "#64748B"}
                strokeWidth="1.5"
              />
            </g>
            <text
              x={humanX + humanW / 2}
              y={humanY + 34}
              textAnchor="middle"
              fill={isHuman ? "#FACC15" : "#A5C5E9"}
              fontSize="10"
              fontWeight={isHuman ? 600 : 400}
            >
              Human
            </text>
            <text
              x={humanX + humanW / 2}
              y={humanY + 44}
              textAnchor="middle"
              fill={isHuman ? "#EAB308" : "#64748B"}
              fontSize="8"
            >
              approves
            </text>
          </g>

          {/* Edge: Wait -> Resume */}
          <path
            d={`M ${waitX + waitW + 2} ${flowY} L ${resumeX - 2} ${flowY}`}
            fill="none"
            stroke={isResumed || isDone ? "rgb(34, 197, 94)" : "#162035"}
            strokeWidth="2"
            className={isResumed || isDone ? "hitl-flow" : ""}
            style={{
              opacity: isResumed || isDone ? 1 : 0.2,
              transition: "all 0.4s ease",
            }}
          />

          {/* Resume box */}
          <rect
            x={resumeX}
            y={flowY - nodeH / 2}
            width={nodeW}
            height={nodeH}
            rx={rx}
            fill={isResumed ? "rgba(34, 197, 94, 0.3)" : "rgba(10, 16, 41, 0.15)"}
            stroke={isResumed ? "url(#hitl-green)" : "#162035"}
            strokeWidth="1.5"
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={resumeX + nodeW / 2}
            y={flowY - 4}
            textAnchor="middle"
            fill={isResumed ? "#86EFAC" : "#64748B"}
            fontSize="12"
            fontWeight="500"
          >
            Resume
          </text>
          <text
            x={resumeX + nodeW / 2}
            y={flowY + 12}
            textAnchor="middle"
            fill={isResumed ? "#4ADE80" : "#1C2B4A"}
            fontSize="9"
          >
            event received
          </text>

          {/* Edge: Resume -> Complete */}
          <path
            d={`M ${resumeX + nodeW + 2} ${flowY} L ${completeX - 2} ${flowY}`}
            fill="none"
            stroke={isDone ? "rgb(34, 197, 94)" : "#162035"}
            strokeWidth="2"
            className={isDone ? "hitl-flow" : ""}
            style={{
              opacity: isDone ? 1 : 0.2,
              transition: "all 0.4s ease",
            }}
          />

          {/* Complete box */}
          <rect
            x={completeX}
            y={flowY - nodeH / 2}
            width={nodeW}
            height={nodeH}
            rx={rx}
            fill={isDone ? "rgba(34, 197, 94, 0.3)" : "rgba(10, 16, 41, 0.15)"}
            stroke={isDone ? "url(#hitl-green)" : "#162035"}
            strokeWidth="1.5"
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={completeX + nodeW / 2}
            y={flowY - 4}
            textAnchor="middle"
            fill={isDone ? "#86EFAC" : "#64748B"}
            fontSize="12"
            fontWeight="500"
          >
            Complete
          </text>
          <text
            x={completeX + nodeW / 2}
            y={flowY + 12}
            textAnchor="middle"
            fill={isDone ? "#4ADE80" : "#1C2B4A"}
            fontSize="9"
          >
            continue
          </text>
        </svg>
      </div>
    </div>
  );
};

export default HumanInLoopDiagram;
