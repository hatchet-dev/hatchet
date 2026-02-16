import React, { useState } from "react";

type Mode = "workflows" | "durable";

const WorkflowComparison: React.FC = () => {
  const [active, setActive] = useState<Mode>("workflows");

  const data = {
    workflows: {
      label: "Workflows (DAGs)",
      color: "indigo",
      items: [
        {
          icon: (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <circle cx="5" cy="10" r="2.5" stroke="#818cf8" strokeWidth="1.5" />
              <circle cx="15" cy="5" r="2.5" stroke="#818cf8" strokeWidth="1.5" />
              <circle cx="15" cy="15" r="2.5" stroke="#818cf8" strokeWidth="1.5" />
              <path d="M7.5 9L12.5 6" stroke="#818cf8" strokeWidth="1.2" />
              <path d="M7.5 11L12.5 14" stroke="#818cf8" strokeWidth="1.2" />
            </svg>
          ),
          title: "Structure",
          desc: "DAG of tasks with declared dependencies",
        },
        {
          icon: (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <rect x="3" y="3" width="14" height="14" rx="2" stroke="#818cf8" strokeWidth="1.5" />
              <path d="M7 10H13" stroke="#818cf8" strokeWidth="1.5" strokeLinecap="round" />
              <path d="M10 7V13" stroke="#818cf8" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          ),
          title: "State",
          desc: "Cached between tasks automatically",
        },
        {
          icon: (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <circle cx="10" cy="10" r="7" stroke="#818cf8" strokeWidth="1.5" />
              <path d="M10 6V10L13 13" stroke="#818cf8" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          ),
          title: "Pausing",
          desc: "Declarative conditions on task definitions",
        },
        {
          icon: (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <path d="M4 10L8 14L16 6" stroke="#818cf8" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          ),
          title: "Recovery",
          desc: "Re-runs failed tasks; completed tasks are skipped",
        },
        {
          icon: (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <rect x="3" y="8" width="5" height="9" rx="1" stroke="#818cf8" strokeWidth="1.5" />
              <rect x="12" y="3" width="5" height="14" rx="1" stroke="#818cf8" strokeWidth="1.5" />
            </svg>
          ),
          title: "Slots",
          desc: "Each task holds a slot while running",
        },
      ],
    },
    durable: {
      label: "Durable Workflows",
      color: "emerald",
      items: [
        {
          icon: (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <path d="M4 4V16H16" stroke="#6ee7b7" strokeWidth="1.5" strokeLinecap="round" />
              <circle cx="8" cy="12" r="1.5" fill="#6ee7b7" />
              <circle cx="12" cy="8" r="1.5" fill="#6ee7b7" />
              <circle cx="15" cy="5" r="1.5" fill="#6ee7b7" />
              <path d="M8 12L12 8L15 5" stroke="#6ee7b7" strokeWidth="1" strokeDasharray="2 2" />
            </svg>
          ),
          title: "Structure",
          desc: "Long-running function with checkpoints",
        },
        {
          icon: (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <rect x="3" y="5" width="14" height="10" rx="2" stroke="#6ee7b7" strokeWidth="1.5" />
              <path d="M7 9H13" stroke="#6ee7b7" strokeWidth="1.5" strokeLinecap="round" />
              <path d="M7 12H11" stroke="#6ee7b7" strokeWidth="1.2" strokeLinecap="round" />
            </svg>
          ),
          title: "State",
          desc: "Stored in a durable event log",
        },
        {
          icon: (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <rect x="7" y="4" width="3" height="12" rx="1" fill="#6ee7b7" />
              <rect x="12" y="4" width="3" height="12" rx="1" fill="#6ee7b7" />
            </svg>
          ),
          title: "Pausing",
          desc: "Inline SleepFor and WaitForEvent calls",
        },
        {
          icon: (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <path d="M6 10C6 7 8 4 10 4C12 4 14 7 14 10" stroke="#6ee7b7" strokeWidth="1.5" strokeLinecap="round" />
              <path d="M14 10L16 8" stroke="#6ee7b7" strokeWidth="1.5" strokeLinecap="round" />
              <path d="M14 10L12 8" stroke="#6ee7b7" strokeWidth="1.5" strokeLinecap="round" />
              <circle cx="10" cy="16" r="1.5" fill="#6ee7b7" />
            </svg>
          ),
          title: "Recovery",
          desc: "Replays from last checkpoint automatically",
        },
        {
          icon: (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <rect x="3" y="8" width="5" height="9" rx="1" stroke="#6ee7b7" strokeWidth="1.5" strokeDasharray="3 2" />
              <rect x="12" y="3" width="5" height="14" rx="1" stroke="#6ee7b7" strokeWidth="1.5" />
              <path d="M5 11L5 14" stroke="#6ee7b7" strokeWidth="1.5" strokeLinecap="round" opacity="0.4" />
            </svg>
          ),
          title: "Slots",
          desc: "Freed during waits — no wasted compute",
        },
      ],
    },
  };

  const current = data[active];
  const isWorkflows = active === "workflows";

  return (
    <div className="my-8 flex flex-col items-center gap-4">
      {/* Toggle */}
      <div className="flex items-center rounded-lg border border-neutral-700/40 bg-neutral-900/50 p-1">
        {(["workflows", "durable"] as Mode[]).map((mode) => (
          <button
            key={mode}
            onClick={() => setActive(mode)}
            className="relative rounded-md px-5 py-2 text-sm font-medium transition-all duration-300"
            style={{
              backgroundColor:
                active === mode
                  ? mode === "workflows"
                    ? "rgba(99,102,241,0.2)"
                    : "rgba(16,185,129,0.2)"
                  : "transparent",
              color:
                active === mode
                  ? mode === "workflows"
                    ? "#c7d2fe"
                    : "#a7f3d0"
                  : "#6b7280",
              border:
                active === mode
                  ? `1px solid ${mode === "workflows" ? "rgba(99,102,241,0.4)" : "rgba(16,185,129,0.4)"}`
                  : "1px solid transparent",
            }}
          >
            {data[mode].label}
          </button>
        ))}
      </div>

      {/* Cards */}
      <div className="w-full max-w-2xl">
        <div className="grid gap-3">
          {current.items.map((item, i) => (
            <div
              key={`${active}-${i}`}
              className="flex items-start gap-4 rounded-lg border px-5 py-4 transition-all duration-300"
              style={{
                backgroundColor: isWorkflows
                  ? "rgba(49,46,129,0.08)"
                  : "rgba(6,78,59,0.08)",
                borderColor: isWorkflows
                  ? "rgba(99,102,241,0.2)"
                  : "rgba(16,185,129,0.2)",
                animationDelay: `${i * 60}ms`,
              }}
            >
              <div
                className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg"
                style={{
                  backgroundColor: isWorkflows
                    ? "rgba(99,102,241,0.15)"
                    : "rgba(16,185,129,0.15)",
                  border: `1px solid ${isWorkflows ? "rgba(99,102,241,0.25)" : "rgba(16,185,129,0.25)"}`,
                }}
              >
                {item.icon}
              </div>
              <div>
                <div
                  className="text-sm font-semibold"
                  style={{
                    color: isWorkflows ? "#c7d2fe" : "#a7f3d0",
                  }}
                >
                  {item.title}
                </div>
                <div className="mt-0.5 text-sm text-gray-400">
                  {item.desc}
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Best-for footer */}
        <div
          className="mt-4 rounded-lg border px-5 py-3 text-center text-sm"
          style={{
            backgroundColor: isWorkflows
              ? "rgba(49,46,129,0.12)"
              : "rgba(6,78,59,0.12)",
            borderColor: isWorkflows
              ? "rgba(99,102,241,0.3)"
              : "rgba(16,185,129,0.3)",
            color: isWorkflows ? "#a5b4fc" : "#86efac",
          }}
        >
          <span className="font-medium">Best for: </span>
          {isWorkflows
            ? "Predictable multi-step pipelines, ETL, CI/CD, and any workflow with a known shape"
            : "Long waits, human-in-the-loop, large fan-outs, and complex procedural logic"}
        </div>
      </div>
    </div>
  );
};

export default WorkflowComparison;
