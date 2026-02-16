import React from "react";

const WorkflowDiagram: React.FC = () => {
  const nodeW = 130;
  const nodeH = 46;
  const rx = 10;

  return (
    <div className="my-8 flex justify-center">
      <div className="w-full max-w-3xl rounded-xl border border-neutral-700/40 bg-neutral-900/30 p-6 backdrop-blur-sm">
        <svg
          viewBox="0 0 720 260"
          className="w-full h-auto"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient id="wf-indigo" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="rgb(99,102,241)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(129,140,248)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="wf-cyan" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="rgb(34,211,238)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(103,232,249)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient id="wf-emerald" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="rgb(16,185,129)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(52,211,153)"
                stopOpacity="0.3"
              />
            </linearGradient>
          </defs>

          <style>{`
            @keyframes wf-dash {
              to { stroke-dashoffset: -20; }
            }
            .wf-flow {
              stroke-dasharray: 8 6;
              animation: wf-dash 0.8s linear infinite;
            }
          `}</style>

          {/* Workflow label */}
          <text
            x="360"
            y="24"
            textAnchor="middle"
            fill="#9ca3af"
            fontSize="11"
            fontWeight="500"
            letterSpacing="0.05em"
          >
            WORKFLOW
          </text>

          {/* Dashed container */}
          <rect
            x="10"
            y="34"
            width="700"
            height="216"
            rx="14"
            fill="none"
            stroke="#444"
            strokeWidth="1"
            strokeDasharray="6 4"
          />

          {/* --- Row 1: Task A → Task B --- */}
          {/* Task A */}
          <rect
            x={40}
            y={60}
            width={nodeW}
            height={nodeH}
            rx={rx}
            fill="rgba(49,46,129,0.3)"
            stroke="url(#wf-indigo)"
            strokeWidth="1.5"
          />
          <text
            x={40 + nodeW / 2}
            y={60 + nodeH / 2 + 1}
            textAnchor="middle"
            fill="#c7d2fe"
            fontSize="12"
            fontWeight="500"
          >
            Task A
          </text>

          {/* Edge A → B */}
          <path
            d={`M ${40 + nodeW + 2} ${60 + nodeH / 2} L ${248} ${60 + nodeH / 2}`}
            fill="none"
            stroke="rgb(129,140,248)"
            strokeWidth="2"
            className="wf-flow"
          />

          {/* Task B */}
          <rect
            x={250}
            y={60}
            width={nodeW}
            height={nodeH}
            rx={rx}
            fill="rgba(49,46,129,0.3)"
            stroke="url(#wf-indigo)"
            strokeWidth="1.5"
          />
          <text
            x={250 + nodeW / 2}
            y={60 + nodeH / 2 + 1}
            textAnchor="middle"
            fill="#c7d2fe"
            fontSize="12"
            fontWeight="500"
          >
            Task B
          </text>

          {/* --- Fan out: B → C and B → D --- */}
          {/* Edge B → C */}
          <path
            d={`M ${250 + nodeW + 2} ${60 + nodeH / 2} C ${430} ${60 + nodeH / 2}, ${430} ${60 + nodeH / 2}, ${460} ${60 + nodeH / 2}`}
            fill="none"
            stroke="rgb(34,211,238)"
            strokeWidth="2"
            className="wf-flow"
            style={{ animationDelay: "0.2s" }}
          />

          {/* Edge B → D */}
          <path
            d={`M ${250 + nodeW + 2} ${60 + nodeH / 2} C ${420} ${60 + nodeH / 2}, ${420} ${170 + nodeH / 2}, ${460} ${170 + nodeH / 2}`}
            fill="none"
            stroke="rgb(34,211,238)"
            strokeWidth="2"
            className="wf-flow"
            style={{ animationDelay: "0.3s" }}
          />

          {/* Task C */}
          <rect
            x={460}
            y={60}
            width={nodeW}
            height={nodeH}
            rx={rx}
            fill="rgba(22,78,99,0.3)"
            stroke="url(#wf-cyan)"
            strokeWidth="1.5"
          />
          <text
            x={460 + nodeW / 2}
            y={60 + nodeH / 2 + 1}
            textAnchor="middle"
            fill="#a5f3fc"
            fontSize="12"
            fontWeight="500"
          >
            Task C
          </text>

          {/* Task D */}
          <rect
            x={460}
            y={170}
            width={nodeW}
            height={nodeH}
            rx={rx}
            fill="rgba(22,78,99,0.3)"
            stroke="url(#wf-cyan)"
            strokeWidth="1.5"
          />
          <text
            x={460 + nodeW / 2}
            y={170 + nodeH / 2 + 1}
            textAnchor="middle"
            fill="#a5f3fc"
            fontSize="12"
            fontWeight="500"
          >
            Task D
          </text>

          {/* --- Both converge to Result --- */}
          {/* Edges C → Result, D → Result drawn before Result box so they don't overlap text */}

          {/* Annotations */}
          <text
            x={40 + nodeW / 2}
            y={60 + nodeH + 16}
            textAnchor="middle"
            fill="#818cf8"
            fontSize="9"
          >
            start
          </text>
          <text
            x={315}
            y={60 + nodeH + 16}
            textAnchor="middle"
            fill="#818cf8"
            fontSize="9"
          >
            depends on A
          </text>
          <text
            x={525}
            y={60 + nodeH + 16}
            textAnchor="middle"
            fill="#67e8f9"
            fontSize="9"
          >
            parallel
          </text>
          <text
            x={525}
            y={170 + nodeH + 16}
            textAnchor="middle"
            fill="#67e8f9"
            fontSize="9"
          >
            parallel
          </text>

          {/* Vertical dashed line between sequential and parallel sections */}
          <line
            x1="430"
            y1="50"
            x2="430"
            y2="240"
            stroke="#444"
            strokeWidth="1"
            strokeDasharray="4 4"
            opacity="0.5"
          />
          <text x="432" y="248" fill="#6b7280" fontSize="8" textAnchor="start">
            fan-out
          </text>

          {/* Labels for sections */}
          <text x="200" y="248" fill="#6b7280" fontSize="8" textAnchor="middle">
            sequential
          </text>
        </svg>
      </div>
    </div>
  );
};

export default WorkflowDiagram;
