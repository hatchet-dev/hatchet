import React from "react";

const PipelineDiagram: React.FC = () => {
  // Layout:
  // Task A          (standalone)
  // Task B -> Task D -> Task E  (pipeline)
  // Task C          (standalone)
  const nodes = [
    { id: "a", label: "Task A", x: 30, y: 40 },
    { id: "b", label: "Task B", x: 30, y: 130 },
    { id: "c", label: "Task C", x: 30, y: 220 },
    { id: "d", label: "Task D", x: 300, y: 130 },
    { id: "e", label: "Task E", x: 560, y: 130 },
  ];

  const nodeWidth = 140;
  const nodeHeight = 50;
  const nodeRx = 10;

  const edges = [
    { from: "b", to: "d" },
    { from: "d", to: "e" },
  ];

  const getNode = (id: string) => nodes.find((n) => n.id === id)!;

  return (
    <div className="my-8 flex justify-center">
      <div className="w-full max-w-3xl rounded-xl border border-neutral-700/40 bg-neutral-900/30 p-6 backdrop-blur-sm">
        <svg
          viewBox="0 0 740 300"
          className="w-full h-auto"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient
              id="pipe-grad"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop
                offset="0%"
                stopColor="rgb(168,85,247)"
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor="rgb(192,132,252)"
                stopOpacity="0.3"
              />
            </linearGradient>
            <linearGradient
              id="pipe-edge-grad"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="0%"
            >
              <stop
                offset="0%"
                stopColor="rgb(168,85,247)"
                stopOpacity="0.8"
              />
              <stop
                offset="100%"
                stopColor="rgb(129,140,248)"
                stopOpacity="0.6"
              />
            </linearGradient>
          </defs>

          <style>{`
            @keyframes pipe-dash {
              to { stroke-dashoffset: -20; }
            }
            .pipe-flow {
              stroke-dasharray: 8 6;
              animation: pipe-dash 0.8s linear infinite;
            }
          `}</style>

          {/* Nodes */}
          {nodes.map((node) => (
            <g key={node.id}>
              <rect
                x={node.x}
                y={node.y}
                width={nodeWidth}
                height={nodeHeight}
                rx={nodeRx}
                fill="rgba(88,28,135,0.25)"
                stroke="url(#pipe-grad)"
                strokeWidth="1.5"
              />
              <text
                x={node.x + nodeWidth / 2}
                y={node.y + nodeHeight / 2 + 5}
                textAnchor="middle"
                fill="#e9d5ff"
                fontSize="13"
                fontWeight="500"
              >
                {node.label}
              </text>
            </g>
          ))}

          {/* Edges (rendered after nodes so lines appear on top) */}
          {edges.map(({ from, to }, i) => {
            const f = getNode(from);
            const t = getNode(to);
            const startX = f.x + nodeWidth + 2;
            const startY = f.y + nodeHeight / 2;
            const endX = t.x - 2;
            const endY = t.y + nodeHeight / 2;
            const midX = (startX + endX) / 2;

            return (
              <path
                key={`edge-${i}`}
                d={`M ${startX} ${startY} C ${midX} ${startY}, ${midX} ${endY}, ${endX} ${endY}`}
                fill="none"
                stroke="rgb(192,132,252)"
                strokeWidth="2"
                className="pipe-flow"
                style={{ animationDelay: `${i * 0.15}s` }}
              />
            );
          })}

          {/* Parallel label and dashed box around A, B, C */}
          <text
            x={30 + nodeWidth / 2}
            y={25}
            textAnchor="middle"
            fill="#9ca3af"
            fontSize="10"
          >
            parallel
          </text>
          <rect
            x={15}
            y={30}
            width={nodeWidth + 30}
            height={255}
            rx={12}
            fill="none"
            stroke="rgb(168,85,247)"
            strokeWidth="1"
            strokeOpacity="0.2"
            strokeDasharray="4 4"
          />
        </svg>
      </div>
    </div>
  );
};

export default PipelineDiagram;
