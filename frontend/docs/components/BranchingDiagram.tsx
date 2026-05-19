import React, { useState } from "react";
import { brand, state, fill, inactive, gradient } from "./diagram-colors";

const BranchingDiagram: React.FC = () => {
  const [isLeft, setIsLeft] = useState(true);

  const nodeWidth = 140;
  const nodeHeight = 50;
  const nodeRx = 10;

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
          style={{ color: leftActive ? state.successLighter : inactive.text }}
        >
          value = 72
        </span>
        <button
          onClick={() => setIsLeft(!isLeft)}
          className="relative h-6 w-11 rounded-full transition-colors duration-300"
          style={{
            backgroundColor: isLeft
              ? "rgba(34, 197, 94, 0.3)"
              : "rgba(51, 146, 255, 0.3)",
            border: `1px solid ${isLeft ? "rgb(34, 197, 94)" : "rgb(133, 189, 255)"}`,
          }}
        >
          <span
            className="absolute top-0.5 h-4 w-4 rounded-full transition-all duration-300"
            style={{
              left: isLeft ? "2px" : "22px",
              backgroundColor: isLeft
                ? "rgb(74, 222, 128)"
                : "rgb(133, 189, 255)",
            }}
          />
        </button>
        <span
          className="text-xs font-medium"
          style={{ color: rightActive ? brand.cyan : inactive.text }}
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
              id="branch-yellow"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
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
            <linearGradient
              id="branch-blue"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
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
            <linearGradient
              id="branch-green"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
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
              fill={fill.activeNode}
              stroke="url(#branch-blue)"
              strokeWidth="1.5"
            />
            <text
              x={30 + nodeWidth / 2}
              y={115 + nodeHeight / 2 + 5}
              textAnchor="middle"
              fill={brand.cyan}
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
              fill={fill.running}
              stroke="url(#branch-yellow)"
              strokeWidth="1.5"
            />
            <text
              x={0}
              y={5}
              textAnchor="middle"
              fill={state.runningLight}
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
              fill={leftActive ? fill.success : fill.activeNode}
              stroke={leftActive ? "url(#branch-green)" : inactive.stroke}
              strokeWidth="1.5"
            />
            <text
              x={410 + nodeWidth / 2}
              y={40 + nodeHeight / 2 - 2}
              textAnchor="middle"
              fill={leftActive ? state.successLighter : inactive.textLight}
              fontSize="12"
              fontWeight="500"
            >
              Task B
            </text>
            <text
              x={410 + nodeWidth / 2}
              y={40 + nodeHeight / 2 + 12}
              textAnchor="middle"
              fill={leftActive ? state.successLight : inactive.text}
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
              fill={fill.activeNode}
              stroke={rightActive ? "url(#branch-blue)" : inactive.stroke}
              strokeWidth="1.5"
            />
            <text
              x={410 + nodeWidth / 2}
              y={190 + nodeHeight / 2 - 2}
              textAnchor="middle"
              fill={rightActive ? brand.cyan : inactive.textLight}
              fontSize="12"
              fontWeight="500"
            >
              Task C
            </text>
            <text
              x={410 + nodeWidth / 2}
              y={190 + nodeHeight / 2 + 12}
              textAnchor="middle"
              fill={rightActive ? brand.blue : inactive.text}
              fontSize="9"
            >
              {rightActive ? "runs ✓" : "skipped"}
            </text>
          </g>

          {/* Task D — always active */}
          <g style={{ opacity: activeOpacity }}>
            <rect
              x={630}
              y={115}
              width={nodeWidth - 20}
              height={nodeHeight}
              rx={nodeRx}
              fill={fill.activeNode}
              stroke="url(#branch-blue)"
              strokeWidth="1.5"
            />
            <text
              x={630 + (nodeWidth - 20) / 2}
              y={115 + nodeHeight / 2 + 5}
              textAnchor="middle"
              fill={brand.cyan}
              fontSize="13"
              fontWeight="500"
            >
              Task D
            </text>
          </g>

          {/* Edge: Task A -> diamond — always active */}
          <path
            d="M 170 140 L 228 140"
            fill="none"
            stroke={gradient.blue[0]}
            strokeWidth="2"
            className="branch-flow"
          />

          {/* Edge: diamond -> Task B */}
          <path
            d="M 310 140 C 350 140, 370 65, 410 65"
            fill="none"
            stroke={leftActive ? gradient.green[0] : inactive.edge}
            strokeWidth="2"
            className={leftActive ? "branch-flow" : ""}
            style={{
              animationDelay: "0.15s",
              opacity: leftActive ? 1 : 0.2,
              transition: "opacity 0.4s ease",
            }}
          />

          {/* Edge: diamond -> Task C */}
          <path
            d="M 310 140 C 350 140, 370 215, 410 215"
            fill="none"
            stroke={rightActive ? gradient.blue[0] : inactive.edge}
            strokeWidth="2"
            className={rightActive ? "branch-flow" : ""}
            style={{
              animationDelay: "0.15s",
              opacity: rightActive ? 1 : 0.2,
              transition: "opacity 0.4s ease",
            }}
          />

          {/* Edge: Task B -> Task D */}
          <path
            d="M 550 65 C 580 65, 600 140, 630 140"
            fill="none"
            stroke={leftActive ? gradient.blue[0] : inactive.edge}
            strokeWidth="2"
            className={leftActive ? "branch-flow" : ""}
            style={{
              animationDelay: "0.3s",
              opacity: leftActive ? 1 : 0.2,
              transition: "opacity 0.4s ease",
            }}
          />

          {/* Edge: Task C -> Task D */}
          <path
            d="M 550 215 C 580 215, 600 140, 630 140"
            fill="none"
            stroke={rightActive ? gradient.blue[0] : inactive.edge}
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
