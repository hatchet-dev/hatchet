import React from "react";

const FanoutDiagram: React.FC = () => {
  const childYPositions = [40, 110, 180, 270];
  const childLabels = ["Child 1", "Child 2", "Child 3", "Child N"];

  return (
    <div className="my-8 flex justify-center">
      <div className="w-full max-w-3xl rounded-xl border border-neutral-700/40 bg-neutral-900/30 p-6 backdrop-blur-sm">
        <svg
          viewBox="0 0 800 330"
          className="w-full h-auto"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient
              id="indigo-grad"
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
            <linearGradient id="cyan-grad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="rgb(34,211,238)" stopOpacity="0.6" />
              <stop
                offset="100%"
                stopColor="rgb(103,232,249)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient
              id="emerald-grad"
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
            @keyframes dash {
              to { stroke-dashoffset: -20; }
            }
            .flow-line-out {
              stroke-dasharray: 8 6;
              animation: dash 0.8s linear infinite;
            }
            .flow-line-in {
              stroke-dasharray: 8 6;
              animation: dash 0.8s linear infinite;
            }
          `}</style>

          {/* Parent Task Box */}
          <rect
            x="20"
            y="125"
            width="170"
            height="80"
            rx="12"
            fill="rgba(49,46,129,0.3)"
            stroke="url(#indigo-grad)"
            strokeWidth="2"
          />
          <text
            x="105"
            y="160"
            textAnchor="middle"
            fill="#c7d2fe"
            fontSize="14"
            fontWeight="600"
          >
            Parent Task
          </text>
          <text
            x="105"
            y="180"
            textAnchor="middle"
            fill="#818cf8"
            fontSize="11"
          >
            spawn(input)
          </text>

          {/* Fan-out lines from parent to children */}
          {childYPositions.map((cy, i) => (
            <path
              key={`out-${i}`}
              d={`M 190 165 C 260 165, 260 ${cy + 25}, 320 ${cy + 25}`}
              fill="none"
              stroke="rgb(99,102,241)"
              strokeWidth="1.5"
              strokeOpacity="0.7"
              className="flow-line-out"
              style={{ animationDelay: `${i * 0.15}s` }}
            />
          ))}

          {/* Child boxes */}
          {childLabels.slice(0, 3).map((label, i) => (
            <g key={label}>
              <rect
                x="320"
                y={childYPositions[i]}
                width="150"
                height="50"
                rx="10"
                fill="rgba(22,78,99,0.3)"
                stroke="url(#cyan-grad)"
                strokeWidth="1.5"
              />
              <text
                x="395"
                y={childYPositions[i] + 30}
                textAnchor="middle"
                fill="#a5f3fc"
                fontSize="12"
                fontWeight="500"
              >
                {label}
              </text>
            </g>
          ))}

          {/* Ellipsis between Child 3 and Child N */}
          <text
            x="395"
            y="255"
            textAnchor="middle"
            fill="#6b7280"
            fontSize="18"
            fontWeight="bold"
          >
            ...
          </text>

          {/* Child N */}
          <rect
            x="320"
            y={childYPositions[3]}
            width="150"
            height="50"
            rx="10"
            fill="rgba(22,78,99,0.3)"
            stroke="url(#cyan-grad)"
            strokeWidth="1.5"
          />
          <text
            x="395"
            y={childYPositions[3] + 30}
            textAnchor="middle"
            fill="#a5f3fc"
            fontSize="12"
            fontWeight="500"
          >
            Child N
          </text>

          {/* Converge lines from children to results */}
          {childYPositions.map((cy, i) => (
            <path
              key={`in-${i}`}
              d={`M 470 ${cy + 25} C 540 ${cy + 25}, 540 165, 600 165`}
              fill="none"
              stroke="rgb(16,185,129)"
              strokeWidth="1.5"
              strokeOpacity="0.7"
              className="flow-line-in"
              style={{ animationDelay: `${i * 0.15 + 0.4}s` }}
            />
          ))}

          {/* Results Box */}
          <rect
            x="600"
            y="125"
            width="180"
            height="80"
            rx="12"
            fill="rgba(6,78,59,0.3)"
            stroke="url(#emerald-grad)"
            strokeWidth="2"
          />
          <text
            x="690"
            y="160"
            textAnchor="middle"
            fill="#a7f3d0"
            fontSize="14"
            fontWeight="600"
          >
            Collect Results
          </text>
          <text
            x="690"
            y="180"
            textAnchor="middle"
            fill="#6ee7b7"
            fontSize="11"
          >
            await all children
          </text>
        </svg>
      </div>
    </div>
  );
};

export default FanoutDiagram;
