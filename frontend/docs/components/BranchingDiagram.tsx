import React, { useState } from "react";

const BranchingDiagram: React.FC = () => {
  const [isLeft, setIsLeft] = useState(true);

  const nodeWidth = 140;
  const nodeHeight = 50;
  const nodeRx = 10;

  // Active vs dimmed styles
  const activeOpacity = 1;
  const dimmedOpacity = 0.2;

  const leftActive = isLeft;
  const rightActive = !isLeft;

  return (
    <div className="my-8 flex flex-col items-center gap-4">
      {/* Toggle */}
      <div className="flex items-center gap-3 rounded-lg border border-neutral-700/40 bg-neutral-900/50 px-4 py-2">
        <span
          className="text-xs font-medium"
          style={{ color: leftActive ? "#a7f3d0" : "#6b7280" }}
        >
          value = 72
        </span>
        <button
          onClick={() => setIsLeft(!isLeft)}
          className="relative h-6 w-11 rounded-full transition-colors duration-300"
          style={{
            backgroundColor: isLeft
              ? "rgba(16,185,129,0.3)"
              : "rgba(99,102,241,0.3)",
            border: `1px solid ${isLeft ? "rgb(16,185,129)" : "rgb(129,140,248)"}`,
          }}
        >
          <span
            className="absolute top-0.5 h-4 w-4 rounded-full transition-all duration-300"
            style={{
              left: isLeft ? "2px" : "22px",
              backgroundColor: isLeft ? "rgb(52,211,153)" : "rgb(129,140,248)",
            }}
          />
        </button>
        <span
          className="text-xs font-medium"
          style={{ color: rightActive ? "#c7d2fe" : "#6b7280" }}
        >
          value = 23
        </span>
      </div>

      {/* Diagram */}
      <div className="w-full max-w-3xl rounded-xl border border-neutral-700/40 bg-neutral-900/30 p-6 backdrop-blur-sm">
        <svg
          viewBox="0 0 780 280"
          className="w-full h-auto"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient
              id="branch-amber"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop offset="0%" stopColor="rgb(245,158,11)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(252,211,77)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient
              id="branch-indigo"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop offset="0%" stopColor="rgb(99,102,241)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(129,140,248)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient
              id="branch-emerald"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop offset="0%" stopColor="rgb(16,185,129)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(52,211,153)"
                stopOpacity="0.3"
              />
            </linearGradient>
          </defs>

          <style>{`
            @keyframes branch-dash {
              to { stroke-dashoffset: -20; }
            }
            .branch-flow {
              stroke-dasharray: 8 6;
              animation: branch-dash 0.8s linear infinite;
            }
          `}</style>

          {/* Task A — always active */}
          <g style={{ opacity: activeOpacity }}>
            <rect
              x={30}
              y={115}
              width={nodeWidth}
              height={nodeHeight}
              rx={nodeRx}
              fill="rgba(49,46,129,0.3)"
              stroke="url(#branch-indigo)"
              strokeWidth="1.5"
            />
            <text
              x={30 + nodeWidth / 2}
              y={115 + nodeHeight / 2 + 5}
              textAnchor="middle"
              fill="#c7d2fe"
              fontSize="13"
              fontWeight="500"
            >
              Task A
            </text>
          </g>

          {/* Condition diamond — always active */}
          <g transform="translate(270, 140)" style={{ opacity: activeOpacity }}>
            <polygon
              points="0,-30 40,0 0,30 -40,0"
              fill="rgba(120,53,15,0.3)"
              stroke="url(#branch-amber)"
              strokeWidth="1.5"
            />
            <text
              x={0}
              y={5}
              textAnchor="middle"
              fill="#fcd34d"
              fontSize="9"
              fontWeight="600"
            >
              {isLeft ? "> 50 ✓" : "≤ 50 ✓"}
            </text>
          </g>

          {/* Left Branch */}
          <g
            style={{
              opacity: leftActive ? activeOpacity : dimmedOpacity,
              transition: "opacity 0.4s ease",
            }}
          >
            <rect
              x={410}
              y={40}
              width={nodeWidth}
              height={nodeHeight}
              rx={nodeRx}
              fill={leftActive ? "rgba(6,78,59,0.3)" : "rgba(30,30,30,0.3)"}
              stroke={leftActive ? "url(#branch-emerald)" : "#555"}
              strokeWidth="1.5"
            />
            <text
              x={410 + nodeWidth / 2}
              y={40 + nodeHeight / 2 - 2}
              textAnchor="middle"
              fill={leftActive ? "#a7f3d0" : "#888"}
              fontSize="12"
              fontWeight="500"
            >
              Left Branch
            </text>
            <text
              x={410 + nodeWidth / 2}
              y={40 + nodeHeight / 2 + 12}
              textAnchor="middle"
              fill={leftActive ? "#6ee7b7" : "#666"}
              fontSize="9"
            >
              {leftActive ? "runs ✓" : "skipped"}
            </text>
          </g>

          {/* Right Branch */}
          <g
            style={{
              opacity: rightActive ? activeOpacity : dimmedOpacity,
              transition: "opacity 0.4s ease",
            }}
          >
            <rect
              x={410}
              y={190}
              width={nodeWidth}
              height={nodeHeight}
              rx={nodeRx}
              fill={rightActive ? "rgba(49,46,129,0.3)" : "rgba(30,30,30,0.3)"}
              stroke={rightActive ? "url(#branch-indigo)" : "#555"}
              strokeWidth="1.5"
            />
            <text
              x={410 + nodeWidth / 2}
              y={190 + nodeHeight / 2 - 2}
              textAnchor="middle"
              fill={rightActive ? "#c7d2fe" : "#888"}
              fontSize="12"
              fontWeight="500"
            >
              Right Branch
            </text>
            <text
              x={410 + nodeWidth / 2}
              y={190 + nodeHeight / 2 + 12}
              textAnchor="middle"
              fill={rightActive ? "#818cf8" : "#666"}
              fontSize="9"
            >
              {rightActive ? "runs ✓" : "skipped"}
            </text>
          </g>

          {/* Task B — always active */}
          <g style={{ opacity: activeOpacity }}>
            <rect
              x={630}
              y={115}
              width={nodeWidth - 20}
              height={nodeHeight}
              rx={nodeRx}
              fill="rgba(49,46,129,0.3)"
              stroke="url(#branch-indigo)"
              strokeWidth="1.5"
            />
            <text
              x={630 + (nodeWidth - 20) / 2}
              y={115 + nodeHeight / 2 + 5}
              textAnchor="middle"
              fill="#c7d2fe"
              fontSize="13"
              fontWeight="500"
            >
              Task B
            </text>
          </g>

          {/* Edge: Task A -> diamond — always active */}
          <path
            d="M 170 140 L 228 140"
            fill="none"
            stroke="rgb(129,140,248)"
            strokeWidth="2"
            className="branch-flow"
          />

          {/* Edge: diamond -> Left Branch */}
          <path
            d="M 310 140 C 350 140, 370 65, 410 65"
            fill="none"
            stroke={leftActive ? "rgb(16,185,129)" : "#444"}
            strokeWidth="2"
            className={leftActive ? "branch-flow" : ""}
            style={{
              animationDelay: "0.15s",
              opacity: leftActive ? 1 : 0.2,
              transition: "opacity 0.4s ease",
            }}
          />

          {/* Edge: diamond -> Right Branch */}
          <path
            d="M 310 140 C 350 140, 370 215, 410 215"
            fill="none"
            stroke={rightActive ? "rgb(129,140,248)" : "#444"}
            strokeWidth="2"
            className={rightActive ? "branch-flow" : ""}
            style={{
              animationDelay: "0.15s",
              opacity: rightActive ? 1 : 0.2,
              transition: "opacity 0.4s ease",
            }}
          />

          {/* Edge: Left Branch -> Task B */}
          <path
            d="M 550 65 C 580 65, 600 140, 630 140"
            fill="none"
            stroke={leftActive ? "rgb(129,140,248)" : "#444"}
            strokeWidth="2"
            className={leftActive ? "branch-flow" : ""}
            style={{
              animationDelay: "0.3s",
              opacity: leftActive ? 1 : 0.2,
              transition: "opacity 0.4s ease",
            }}
          />

          {/* Edge: Right Branch -> Task B */}
          <path
            d="M 550 215 C 580 215, 600 140, 630 140"
            fill="none"
            stroke={rightActive ? "rgb(129,140,248)" : "#444"}
            strokeWidth="2"
            className={rightActive ? "branch-flow" : ""}
            style={{
              animationDelay: "0.3s",
              opacity: rightActive ? 1 : 0.2,
              transition: "opacity 0.4s ease",
            }}
          />
        </svg>
      </div>
    </div>
  );
};

export default BranchingDiagram;
