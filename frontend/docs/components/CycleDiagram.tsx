import React, { useState, useEffect } from "react";

const CycleDiagram: React.FC = () => {
  const [iteration, setIteration] = useState(0);
  const maxIterations = 3;

  useEffect(() => {
    const timer = setInterval(() => {
      setIteration((prev) => (prev + 1) % (maxIterations + 1));
    }, 2000);
    return () => clearInterval(timer);
  }, []);

  const nodeWidth = 140;
  const nodeHeight = 50;
  const nodeRx = 10;

  // Positions
  const taskX = 60;
  const taskY = 100;
  const checkX = 300;
  const checkY = 125;
  const doneX = 520;
  const doneY = 100;

  const isDone = iteration === maxIterations;

  return (
    <div className="my-8 flex flex-col items-center gap-4">
      {/* Iteration counter */}
      <div className="flex items-center gap-3 rounded-lg border border-neutral-700/40 bg-neutral-900/50 px-4 py-2">
        <span className="text-xs font-medium text-gray-400">Iteration:</span>
        {[0, 1, 2].map((i) => (
          <span
            key={i}
            className="flex h-6 w-6 items-center justify-center rounded-full text-xs font-bold transition-all duration-300"
            style={{
              backgroundColor:
                i < iteration
                  ? "rgba(16,185,129,0.3)"
                  : i === iteration && !isDone
                    ? "rgba(245,158,11,0.3)"
                    : "rgba(50,50,50,0.3)",
              border: `1px solid ${
                i < iteration
                  ? "rgb(16,185,129)"
                  : i === iteration && !isDone
                    ? "rgb(245,158,11)"
                    : "#555"
              }`,
              color:
                i < iteration
                  ? "#6ee7b7"
                  : i === iteration && !isDone
                    ? "#fcd34d"
                    : "#666",
            }}
          >
            {i + 1}
          </span>
        ))}
        <span
          className="ml-1 text-xs font-medium transition-colors duration-300"
          style={{ color: isDone ? "#6ee7b7" : "#6b7280" }}
        >
          {isDone ? "done!" : "running..."}
        </span>
      </div>

      {/* Diagram */}
      <div className="w-full max-w-3xl rounded-xl border border-neutral-700/40 bg-neutral-900/30 p-6 backdrop-blur-sm">
        <svg
          viewBox="0 0 700 230"
          className="w-full h-auto"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient
              id="cycle-amber"
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
              id="cycle-indigo"
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
              id="cycle-emerald"
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
            @keyframes cycle-dash {
              to { stroke-dashoffset: -20; }
            }
            .cycle-flow {
              stroke-dasharray: 8 6;
              animation: cycle-dash 0.8s linear infinite;
            }
          `}</style>

          {/* Task box */}
          <rect
            x={taskX}
            y={taskY}
            width={nodeWidth}
            height={nodeHeight}
            rx={nodeRx}
            fill={!isDone ? "rgba(49,46,129,0.3)" : "rgba(49,46,129,0.15)"}
            stroke={!isDone ? "url(#cycle-indigo)" : "#555"}
            strokeWidth="1.5"
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={taskX + nodeWidth / 2}
            y={taskY + nodeHeight / 2 - 2}
            textAnchor="middle"
            fill={!isDone ? "#c7d2fe" : "#888"}
            fontSize="13"
            fontWeight="500"
          >
            Task
          </text>
          <text
            x={taskX + nodeWidth / 2}
            y={taskY + nodeHeight / 2 + 14}
            textAnchor="middle"
            fill={!isDone ? "#818cf8" : "#666"}
            fontSize="9"
          >
            do work
          </text>

          {/* Condition diamond */}
          <g transform={`translate(${checkX}, ${checkY})`}>
            <polygon
              points="0,-30 45,0 0,30 -45,0"
              fill="rgba(120,53,15,0.3)"
              stroke="url(#cycle-amber)"
              strokeWidth="1.5"
            />
            <text
              x={0}
              y={-2}
              textAnchor="middle"
              fill="#fcd34d"
              fontSize="9"
              fontWeight="600"
            >
              {isDone ? "done ✓" : "done?"}
            </text>
            <text x={0} y={12} textAnchor="middle" fill="#d97706" fontSize="8">
              {isDone ? "" : "not yet"}
            </text>
          </g>

          {/* Done box */}
          <rect
            x={doneX}
            y={doneY}
            width={nodeWidth}
            height={nodeHeight}
            rx={nodeRx}
            fill={isDone ? "rgba(6,78,59,0.3)" : "rgba(30,30,30,0.15)"}
            stroke={isDone ? "url(#cycle-emerald)" : "#444"}
            strokeWidth="1.5"
            style={{ transition: "all 0.4s ease" }}
          />
          <text
            x={doneX + nodeWidth / 2}
            y={doneY + nodeHeight / 2 - 2}
            textAnchor="middle"
            fill={isDone ? "#a7f3d0" : "#666"}
            fontSize="13"
            fontWeight="500"
            style={{ transition: "fill 0.4s ease" }}
          >
            Complete
          </text>
          <text
            x={doneX + nodeWidth / 2}
            y={doneY + nodeHeight / 2 + 14}
            textAnchor="middle"
            fill={isDone ? "#6ee7b7" : "#555"}
            fontSize="9"
            style={{ transition: "fill 0.4s ease" }}
          >
            return result
          </text>

          {/* Edge: Task -> Check */}
          <path
            d={`M ${taskX + nodeWidth + 2} ${taskY + nodeHeight / 2} L ${checkX - 47} ${checkY}`}
            fill="none"
            stroke={!isDone ? "rgb(129,140,248)" : "#555"}
            strokeWidth="2"
            className={!isDone ? "cycle-flow" : ""}
            style={{ transition: "stroke 0.4s ease" }}
          />

          {/* Edge: Check -> Done (right) */}
          <path
            d={`M ${checkX + 47} ${checkY} L ${doneX - 2} ${doneY + nodeHeight / 2}`}
            fill="none"
            stroke={isDone ? "rgb(16,185,129)" : "#444"}
            strokeWidth="2"
            className={isDone ? "cycle-flow" : ""}
            style={{
              opacity: isDone ? 1 : 0.2,
              transition: "all 0.4s ease",
            }}
          />
          {/* "yes" label on done edge */}
          <text
            x={(checkX + 47 + doneX - 2) / 2}
            y={doneY + nodeHeight / 2 - 10}
            textAnchor="middle"
            fill={isDone ? "#6ee7b7" : "#555"}
            fontSize="9"
            style={{ transition: "fill 0.4s ease" }}
          >
            yes
          </text>

          {/* Loop-back edge: Check -> Task (curved below) */}
          <path
            d={`M ${checkX} ${checkY + 30} C ${checkX} ${checkY + 80}, ${taskX + nodeWidth / 2} ${taskY + nodeHeight + 50}, ${taskX + nodeWidth / 2} ${taskY + nodeHeight + 2}`}
            fill="none"
            stroke={!isDone ? "rgb(245,158,11)" : "#444"}
            strokeWidth="2"
            className={!isDone ? "cycle-flow" : ""}
            style={{
              opacity: !isDone ? 1 : 0.2,
              transition: "all 0.4s ease",
            }}
          />
          {/* "no, loop" label */}
          <text
            x={(checkX + taskX + nodeWidth / 2) / 2}
            y={checkY + 75}
            textAnchor="middle"
            fill={!isDone ? "#fcd34d" : "#555"}
            fontSize="9"
            style={{ transition: "fill 0.4s ease" }}
          >
            no → re-run
          </text>
        </svg>
      </div>
    </div>
  );
};

export default CycleDiagram;
