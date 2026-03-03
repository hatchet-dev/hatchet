import React, { useState, useEffect } from "react";
import { brand, state, fill, inactive, gradient } from "./diagram-colors";

const LongWaitDiagram: React.FC = () => {
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

  const waitLabel = "Sleep 24h";
  const waitSublabel = "durable pause";
  const triggerLabel = "time elapsed";

  return (
    <div className="my-8 flex flex-col items-center gap-4">
      {/* Status */}
      <div className="flex items-center gap-2 rounded-lg border border-neutral-700/40 bg-neutral-900/50 px-4 py-2">
        <span
          className="text-xs transition-colors duration-300"
          style={{
            color: isDone
              ? state.successLight
              : isWaiting
                ? brand.cyan
                : state.queued,
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
            <linearGradient id="lw-blue" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor={gradient.blue[0]} stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor={gradient.blue[1]}
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="lw-yellow" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor={gradient.yellow[0]} stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor={gradient.yellow[1]}
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="lw-green" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor={gradient.green[0]} stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor={gradient.green[1]}
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
            fill={phase === 0 ? fill.activeNode : fill.inactiveNode}
            stroke={phase === 0 ? "url(#lw-blue)" : inactive.stroke}
            strokeWidth="1.5"
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={taskX + nodeW / 2}
            y={taskY + nodeH / 2 - 2}
            textAnchor="middle"
            fill={phase === 0 ? brand.cyan : inactive.textLight}
            fontSize="13"
            fontWeight="500"
          >
            Task Runs
          </text>
          <text
            x={taskX + nodeW / 2}
            y={taskY + nodeH / 2 + 14}
            textAnchor="middle"
            fill={phase === 0 ? brand.blue : inactive.text}
            fontSize="9"
          >
            do work
          </text>

          {/* Edge: Task -> Wait */}
          <path
            d={`M ${taskX + nodeW + 2} ${taskY + nodeH / 2} L ${waitX - 2} ${waitY + nodeH / 2}`}
            fill="none"
            stroke={phase === 0 ? gradient.blue[0] : inactive.stroke}
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
                ? "rgba(10, 16, 41, 0.25)"
                : fill.inactiveNode
            }
            stroke={
              isWaiting
                ? "url(#lw-blue)"
                : inactive.stroke
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
                fill={brand.blue}
              />
              <rect
                x={waitX + (nodeW + 60) / 2 + 4}
                y={waitY + 28}
                width="5"
                height="16"
                rx="1"
                fill={brand.blue}
              />
            </g>
          )}
          <text
            x={waitX + (nodeW + 60) / 2}
            y={waitY + 10}
            textAnchor="middle"
            fill={isWaiting ? brand.cyan : inactive.textLight}
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
              fill={inactive.text}
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
            fill={isResumed || isDone ? state.successLight : inactive.stroke}
            fontSize="9"
            style={{ transition: "fill 0.4s ease" }}
          >
            {isResumed || isDone ? triggerLabel : ""}
          </text>

          {/* Edge: Wait -> Resume */}
          <path
            d={`M ${waitX + nodeW + 62} ${waitY + nodeH / 2} L ${resumeX - 2} ${resumeY + nodeH / 2}`}
            fill="none"
            stroke={isResumed ? gradient.green[0] : inactive.edge}
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
            fill={isResumed ? fill.success : fill.inactiveNode}
            stroke={isResumed ? "url(#lw-green)" : inactive.edge}
            strokeWidth="1.5"
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={resumeX + nodeW / 2}
            y={resumeY + nodeH / 2 - 2}
            textAnchor="middle"
            fill={isResumed ? state.successLighter : inactive.text}
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
            fill={isResumed ? state.successLight : inactive.stroke}
            fontSize="9"
            style={{ transition: "fill 0.4s ease" }}
          >
            continue work
          </text>

          {/* Edge: Resume -> Complete */}
          <path
            d={`M ${resumeX + nodeW + 2} ${resumeY + nodeH / 2} L ${completeX - 2} ${completeY + nodeH / 2}`}
            fill="none"
            stroke={isDone ? gradient.green[0] : inactive.edge}
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
            fill={isDone ? fill.success : fill.inactiveNode}
            stroke={isDone ? "url(#lw-green)" : inactive.edge}
            strokeWidth="1.5"
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={completeX + nodeW / 2}
            y={completeY + nodeH / 2 - 2}
            textAnchor="middle"
            fill={isDone ? state.successLighter : inactive.text}
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
            fill={isDone ? state.successLight : inactive.stroke}
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
