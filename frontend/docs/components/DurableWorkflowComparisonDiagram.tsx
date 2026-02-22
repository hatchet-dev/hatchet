import React from "react";

const DurableWorkflowComparisonDiagram: React.FC = () => {
  const nodeW = 130;
  const nodeH = 36;
  const smallW = 100;
  const smallH = 30;
  const rx = 8;

  return (
    <div className="my-8 flex justify-center">
      <div className="w-full max-w-4xl rounded-xl border border-neutral-700/40 bg-neutral-900/30 p-6 backdrop-blur-sm">
        <div className="grid grid-cols-2 gap-6">
          {/* Left: Durable Task Execution */}
          <div>
            <svg
              viewBox="0 0 340 400"
              className="w-full h-auto"
              xmlns="http://www.w3.org/2000/svg"
            >
              <defs>
                <linearGradient
                  id="dc-indigo"
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
                  id="dc-amber"
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
                  id="dc-cyan"
                  x1="0%"
                  y1="0%"
                  x2="100%"
                  y2="100%"
                >
                  <stop
                    offset="0%"
                    stopColor="rgb(34,211,238)"
                    stopOpacity="0.6"
                  />
                  <stop
                    offset="100%"
                    stopColor="rgb(103,232,249)"
                    stopOpacity="0.3"
                  />
                </linearGradient>
              </defs>

              <style>{`
                @keyframes dc-dash {
                  to { stroke-dashoffset: -20; }
                }
                .dc-flow {
                  stroke-dasharray: 8 6;
                  animation: dc-dash 0.8s linear infinite;
                }
              `}</style>

              <text
                x="170"
                y="20"
                textAnchor="middle"
                fill="#c7d2fe"
                fontSize="13"
                fontWeight="600"
              >
                Durable Task
              </text>
              <text
                x="170"
                y="34"
                textAnchor="middle"
                fill="#818cf8"
                fontSize="10"
              >
                shape of work is dynamic
              </text>

              {/* Container */}
              <rect
                x="10"
                y="46"
                width="320"
                height="344"
                rx="12"
                fill="none"
                stroke="#444"
                strokeWidth="1"
                strokeDasharray="6 4"
              />

              {/* Step 1: Do work */}
              <rect
                x={30}
                y={64}
                width={nodeW}
                height={nodeH}
                rx={rx}
                fill="rgba(49,46,129,0.3)"
                stroke="url(#dc-indigo)"
                strokeWidth="1.5"
              />
              <text
                x={30 + nodeW / 2}
                y={64 + nodeH / 2 - 4}
                textAnchor="middle"
                fill="#c7d2fe"
                fontSize="11"
                fontWeight="500"
              >
                do_work()
              </text>
              <text
                x={30 + nodeW / 2}
                y={64 + nodeH / 2 + 8}
                textAnchor="middle"
                fill="#818cf8"
                fontSize="9"
              >
                line 12
              </text>

              {/* Arrow 1→2 */}
              <line
                x1={30 + nodeW / 2}
                y1={64 + nodeH}
                x2={30 + nodeW / 2}
                y2={64 + nodeH + 18}
                stroke="rgb(129,140,248)"
                strokeWidth="2"
                className="dc-flow"
              />

              {/* Step 2: sleep_for (checkpoint) */}
              <rect
                x={30}
                y={118}
                width={nodeW}
                height={nodeH}
                rx={rx}
                fill="rgba(120,53,15,0.25)"
                stroke="url(#dc-amber)"
                strokeWidth="1.5"
                strokeDasharray="4 3"
              />
              <text
                x={30 + nodeW / 2}
                y={118 + nodeH / 2 - 4}
                textAnchor="middle"
                fill="#fcd34d"
                fontSize="11"
                fontWeight="500"
              >
                sleep_for(24h)
              </text>
              <text
                x={30 + nodeW / 2}
                y={118 + nodeH / 2 + 8}
                textAnchor="middle"
                fill="#d97706"
                fontSize="9"
              >
                checkpoint
              </text>
              {/* Save icon */}
              <svg
                x={30 + nodeW + 6}
                y={120}
                width="13"
                height="13"
                viewBox="0 0 24 24"
                fill="none"
                stroke="#fcd34d"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
                <polyline points="17 21 17 13 7 13 7 21" />
                <polyline points="7 3 7 8 15 8" />
              </svg>

              {/* Arrow 2→3 */}
              <line
                x1={30 + nodeW / 2}
                y1={118 + nodeH}
                x2={30 + nodeW / 2}
                y2={118 + nodeH + 18}
                stroke="rgb(129,140,248)"
                strokeWidth="2"
                className="dc-flow"
              />

              {/* Step 3: spawn_tasks */}
              <rect
                x={30}
                y={172}
                width={nodeW}
                height={nodeH}
                rx={rx}
                fill="rgba(22,78,99,0.3)"
                stroke="url(#dc-cyan)"
                strokeWidth="1.5"
              />
              <text
                x={30 + nodeW / 2}
                y={172 + nodeH / 2 - 4}
                textAnchor="middle"
                fill="#a5f3fc"
                fontSize="11"
                fontWeight="500"
              >
                spawn_tasks()
              </text>
              <text
                x={30 + nodeW / 2}
                y={172 + nodeH / 2 + 8}
                textAnchor="middle"
                fill="#67e8f9"
                fontSize="9"
              >
                fan-out
              </text>

              {/* Spawn arrows to child tasks */}
              <path
                d={`M ${30 + nodeW} ${172 + nodeH / 2} C ${30 + nodeW + 30} ${172 + nodeH / 2}, ${210} ${172}, ${220} ${172}`}
                fill="none"
                stroke="rgb(34,211,238)"
                strokeWidth="1.5"
                className="dc-flow"
                style={{ animationDelay: "0.1s" }}
              />
              <path
                d={`M ${30 + nodeW} ${172 + nodeH / 2} C ${30 + nodeW + 30} ${172 + nodeH / 2}, ${210} ${212}, ${220} ${212}`}
                fill="none"
                stroke="rgb(34,211,238)"
                strokeWidth="1.5"
                className="dc-flow"
                style={{ animationDelay: "0.2s" }}
              />

              {/* Child task A */}
              <rect
                x={220}
                y={158}
                width={smallW}
                height={smallH}
                rx={6}
                fill="rgba(22,78,99,0.2)"
                stroke="rgb(34,211,238)"
                strokeWidth="1"
              />
              <text
                x={220 + smallW / 2}
                y={158 + smallH / 2 + 1}
                textAnchor="middle"
                fill="#a5f3fc"
                fontSize="10"
                fontWeight="500"
              >
                child task 1
              </text>

              {/* Child task B */}
              <rect
                x={220}
                y={198}
                width={smallW}
                height={smallH}
                rx={6}
                fill="rgba(22,78,99,0.2)"
                stroke="rgb(34,211,238)"
                strokeWidth="1"
              />
              <text
                x={220 + smallW / 2}
                y={198 + smallH / 2 + 1}
                textAnchor="middle"
                fill="#a5f3fc"
                fontSize="10"
                fontWeight="500"
              >
                child task 2
              </text>

              {/* "..." more children */}
              <text
                x={220 + smallW / 2}
                y={242}
                textAnchor="middle"
                fill="#67e8f9"
                fontSize="11"
              >
                ...
              </text>

              {/* Arrow 3→4 */}
              <line
                x1={30 + nodeW / 2}
                y1={172 + nodeH}
                x2={30 + nodeW / 2}
                y2={172 + nodeH + 18}
                stroke="rgb(129,140,248)"
                strokeWidth="2"
                className="dc-flow"
              />

              {/* Step 4: collect results (checkpoint) */}
              <rect
                x={30}
                y={226}
                width={nodeW}
                height={nodeH}
                rx={rx}
                fill="rgba(120,53,15,0.25)"
                stroke="url(#dc-amber)"
                strokeWidth="1.5"
                strokeDasharray="4 3"
              />
              <text
                x={30 + nodeW / 2}
                y={226 + nodeH / 2 - 4}
                textAnchor="middle"
                fill="#fcd34d"
                fontSize="11"
                fontWeight="500"
              >
                wait_for_results()
              </text>
              <text
                x={30 + nodeW / 2}
                y={226 + nodeH / 2 + 8}
                textAnchor="middle"
                fill="#d97706"
                fontSize="9"
              >
                checkpoint
              </text>
              {/* Save icon */}
              <svg
                x={30 + nodeW + 6}
                y={228}
                width="13"
                height="13"
                viewBox="0 0 24 24"
                fill="none"
                stroke="#fcd34d"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
                <polyline points="17 21 17 13 7 13 7 21" />
                <polyline points="7 3 7 8 15 8" />
              </svg>

              {/* Arrow 4→5 */}
              <line
                x1={30 + nodeW / 2}
                y1={226 + nodeH}
                x2={30 + nodeW / 2}
                y2={226 + nodeH + 18}
                stroke="rgb(129,140,248)"
                strokeWidth="2"
                className="dc-flow"
              />

              {/* Step 5: process results */}
              <rect
                x={30}
                y={280}
                width={nodeW}
                height={nodeH}
                rx={rx}
                fill="rgba(49,46,129,0.3)"
                stroke="url(#dc-indigo)"
                strokeWidth="1.5"
              />
              <text
                x={30 + nodeW / 2}
                y={280 + nodeH / 2 - 4}
                textAnchor="middle"
                fill="#c7d2fe"
                fontSize="11"
                fontWeight="500"
              >
                process_results()
              </text>
              <text
                x={30 + nodeW / 2}
                y={280 + nodeH / 2 + 8}
                textAnchor="middle"
                fill="#818cf8"
                fontSize="9"
              >
                line 20
              </text>

              {/* Left annotation: call stack bracket */}
              <line
                x1={18}
                y1={64}
                x2={18}
                y2={280 + nodeH}
                stroke="#555"
                strokeWidth="1"
              />
              <line
                x1={18}
                y1={64}
                x2={24}
                y2={64}
                stroke="#555"
                strokeWidth="1"
              />
              <line
                x1={18}
                y1={280 + nodeH}
                x2={24}
                y2={280 + nodeH}
                stroke="#555"
                strokeWidth="1"
              />
              <text
                x={12}
                y={190}
                textAnchor="middle"
                fill="#6b7280"
                fontSize="9"
                transform="rotate(-90, 12, 190)"
              >
                single function
              </text>

              {/* Annotation: children run independently */}
              <text
                x={220 + smallW / 2}
                y={255}
                textAnchor="middle"
                fill="#6b7280"
                fontSize="8"
              >
                run on any worker
              </text>

              {/* Bottom annotation */}
              <text
                x={170}
                y={338}
                textAnchor="middle"
                fill="#6b7280"
                fontSize="8"
              >
                procedural · checkpoints · N decided at runtime
              </text>
            </svg>
          </div>

          {/* Right: DAG */}
          <div>
            <svg
              viewBox="0 0 340 400"
              className="w-full h-auto"
              xmlns="http://www.w3.org/2000/svg"
            >
              <defs>
                <linearGradient
                  id="dg-indigo"
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
                  id="dg-cyan"
                  x1="0%"
                  y1="0%"
                  x2="100%"
                  y2="100%"
                >
                  <stop
                    offset="0%"
                    stopColor="rgb(34,211,238)"
                    stopOpacity="0.6"
                  />
                  <stop
                    offset="100%"
                    stopColor="rgb(103,232,249)"
                    stopOpacity="0.3"
                  />
                </linearGradient>
                <linearGradient
                  id="dg-emerald"
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
                @keyframes dg-dash {
                  to { stroke-dashoffset: -20; }
                }
                .dg-flow {
                  stroke-dasharray: 8 6;
                  animation: dg-dash 0.8s linear infinite;
                }
              `}</style>

              <text
                x="170"
                y="20"
                textAnchor="middle"
                fill="#c7d2fe"
                fontSize="13"
                fontWeight="600"
              >
                DAG Workflow
              </text>
              <text
                x="170"
                y="34"
                textAnchor="middle"
                fill="#818cf8"
                fontSize="10"
              >
                shape of work is known upfront
              </text>

              {/* Container */}
              <rect
                x="10"
                y="46"
                width="320"
                height="344"
                rx="12"
                fill="none"
                stroke="#444"
                strokeWidth="1"
                strokeDasharray="6 4"
              />

              {/* Task A (top) */}
              <rect
                x={105}
                y={80}
                width={nodeW}
                height={nodeH}
                rx={rx}
                fill="rgba(49,46,129,0.3)"
                stroke="url(#dg-indigo)"
                strokeWidth="1.5"
              />
              <text
                x={105 + nodeW / 2}
                y={80 + nodeH / 2 + 1}
                textAnchor="middle"
                fill="#c7d2fe"
                fontSize="11"
                fontWeight="500"
              >
                Extract
              </text>

              {/* Fan out: A → B and A → C */}
              <path
                d={`M ${170} ${80 + nodeH} C ${170} ${80 + nodeH + 30}, ${85} ${164}, ${85} ${170}`}
                fill="none"
                stroke="rgb(34,211,238)"
                strokeWidth="2"
                className="dg-flow"
              />
              <path
                d={`M ${170} ${80 + nodeH} C ${170} ${80 + nodeH + 30}, ${255} ${164}, ${255} ${170}`}
                fill="none"
                stroke="rgb(34,211,238)"
                strokeWidth="2"
                className="dg-flow"
                style={{ animationDelay: "0.15s" }}
              />

              {/* Task B (left) */}
              <rect
                x={20}
                y={170}
                width={nodeW}
                height={nodeH}
                rx={rx}
                fill="rgba(22,78,99,0.3)"
                stroke="url(#dg-cyan)"
                strokeWidth="1.5"
              />
              <text
                x={20 + nodeW / 2}
                y={170 + nodeH / 2 + 1}
                textAnchor="middle"
                fill="#a5f3fc"
                fontSize="11"
                fontWeight="500"
              >
                Transform A
              </text>

              {/* Task C (right) */}
              <rect
                x={190}
                y={170}
                width={nodeW}
                height={nodeH}
                rx={rx}
                fill="rgba(22,78,99,0.3)"
                stroke="url(#dg-cyan)"
                strokeWidth="1.5"
              />
              <text
                x={190 + nodeW / 2}
                y={170 + nodeH / 2 + 1}
                textAnchor="middle"
                fill="#a5f3fc"
                fontSize="11"
                fontWeight="500"
              >
                Transform B
              </text>

              {/* Fan in: B → D and C → D */}
              <path
                d={`M ${85} ${170 + nodeH} C ${85} ${170 + nodeH + 30}, ${170} ${254}, ${170} ${260}`}
                fill="none"
                stroke="rgb(16,185,129)"
                strokeWidth="2"
                className="dg-flow"
                style={{ animationDelay: "0.3s" }}
              />
              <path
                d={`M ${255} ${170 + nodeH} C ${255} ${170 + nodeH + 30}, ${170} ${254}, ${170} ${260}`}
                fill="none"
                stroke="rgb(16,185,129)"
                strokeWidth="2"
                className="dg-flow"
                style={{ animationDelay: "0.4s" }}
              />

              {/* Task D (bottom, merge) */}
              <rect
                x={105}
                y={260}
                width={nodeW}
                height={nodeH}
                rx={rx}
                fill="rgba(6,78,59,0.3)"
                stroke="url(#dg-emerald)"
                strokeWidth="1.5"
              />
              <text
                x={105 + nodeW / 2}
                y={260 + nodeH / 2 + 1}
                textAnchor="middle"
                fill="#a7f3d0"
                fontSize="11"
                fontWeight="500"
              >
                Load
              </text>

              {/* Annotations */}
              <text
                x={105 + nodeW / 2}
                y={80 + nodeH + 14}
                textAnchor="middle"
                fill="#818cf8"
                fontSize="8"
              >
                start
              </text>
              <text
                x={170}
                y={152}
                textAnchor="middle"
                fill="#67e8f9"
                fontSize="8"
              >
                parallel
              </text>
              <text
                x={170}
                y={254}
                textAnchor="middle"
                fill="#6ee7b7"
                fontSize="8"
              >
                waits for both
              </text>

              {/* Bottom annotation */}
              <text
                x={170}
                y={338}
                textAnchor="middle"
                fill="#6b7280"
                fontSize="8"
              >
                declared graph · fixed shape · each task independent
              </text>
            </svg>
          </div>
        </div>
      </div>
    </div>
  );
};

export default DurableWorkflowComparisonDiagram;
