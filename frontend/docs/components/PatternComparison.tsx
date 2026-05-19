import React from "react";

interface ComparisonRow {
  label: string;
  workflow: string;
  durable: string;
}

interface PatternComparisonProps {
  rows: ComparisonRow[];
  recommendation?: "workflow" | "durable" | "both";
  recommendationText?: string;
}

const PatternComparison: React.FC<PatternComparisonProps> = ({
  rows,
  recommendation = "workflow",
  recommendationText,
}) => {
  return (
    <div className="my-6">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {/* Workflows column */}
        <div
          className="rounded-xl border p-5"
          style={{
            borderColor: "rgba(99,102,241,0.25)",
            backgroundColor: "rgba(49,46,129,0.06)",
          }}
        >
          <div className="mb-4 flex items-center gap-2.5">
            <div
              className="flex h-8 w-8 items-center justify-center rounded-lg"
              style={{
                backgroundColor: "rgba(99,102,241,0.15)",
                border: "1px solid rgba(99,102,241,0.3)",
              }}
            >
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <circle
                  cx="4"
                  cy="8"
                  r="2"
                  stroke="#818cf8"
                  strokeWidth="1.5"
                />
                <circle
                  cx="12"
                  cy="4"
                  r="2"
                  stroke="#818cf8"
                  strokeWidth="1.5"
                />
                <circle
                  cx="12"
                  cy="12"
                  r="2"
                  stroke="#818cf8"
                  strokeWidth="1.5"
                />
                <path d="M6 7L10 5" stroke="#818cf8" strokeWidth="1" />
                <path d="M6 9L10 11" stroke="#818cf8" strokeWidth="1" />
              </svg>
            </div>
            <span
              className="text-sm font-semibold"
              style={{ color: "#c7d2fe" }}
            >
              Workflows (DAGs)
            </span>
            {recommendation === "workflow" && (
              <span
                className="ml-auto rounded-full px-2 py-0.5 text-[10px] font-medium"
                style={{
                  backgroundColor: "rgba(99,102,241,0.2)",
                  color: "#a5b4fc",
                  border: "1px solid rgba(99,102,241,0.3)",
                }}
              >
                recommended
              </span>
            )}
          </div>
          <div className="flex flex-col gap-3">
            {rows.map((row, i) => (
              <div key={i}>
                <div
                  className="mb-1 text-[11px] font-medium uppercase tracking-wider"
                  style={{ color: "#818cf8" }}
                >
                  {row.label}
                </div>
                <div className="text-sm text-gray-300">{row.workflow}</div>
              </div>
            ))}
          </div>
        </div>

        {/* Durable column */}
        <div
          className="rounded-xl border p-5"
          style={{
            borderColor: "rgba(16,185,129,0.25)",
            backgroundColor: "rgba(6,78,59,0.06)",
          }}
        >
          <div className="mb-4 flex items-center gap-2.5">
            <div
              className="flex h-8 w-8 items-center justify-center rounded-lg"
              style={{
                backgroundColor: "rgba(16,185,129,0.15)",
                border: "1px solid rgba(16,185,129,0.3)",
              }}
            >
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path
                  d="M3 3v10h10"
                  stroke="#6ee7b7"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                />
                <circle cx="6" cy="10" r="1.5" fill="#6ee7b7" />
                <circle cx="10" cy="6" r="1.5" fill="#6ee7b7" />
                <path
                  d="M6 10L10 6"
                  stroke="#6ee7b7"
                  strokeWidth="1"
                  strokeDasharray="2 2"
                />
              </svg>
            </div>
            <span
              className="text-sm font-semibold"
              style={{ color: "#a7f3d0" }}
            >
              Durable Workflows
            </span>
            {recommendation === "durable" && (
              <span
                className="ml-auto rounded-full px-2 py-0.5 text-[10px] font-medium"
                style={{
                  backgroundColor: "rgba(16,185,129,0.2)",
                  color: "#86efac",
                  border: "1px solid rgba(16,185,129,0.3)",
                }}
              >
                recommended
              </span>
            )}
          </div>
          <div className="flex flex-col gap-3">
            {rows.map((row, i) => (
              <div key={i}>
                <div
                  className="mb-1 text-[11px] font-medium uppercase tracking-wider"
                  style={{ color: "#6ee7b7" }}
                >
                  {row.label}
                </div>
                <div className="text-sm text-gray-300">{row.durable}</div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Recommendation footer */}
      {recommendationText && (
        <div
          className="mt-3 rounded-lg border px-4 py-2.5 text-center text-sm"
          style={{
            borderColor:
              recommendation === "durable"
                ? "rgba(16,185,129,0.2)"
                : recommendation === "both"
                  ? "rgba(99,102,241,0.15)"
                  : "rgba(99,102,241,0.2)",
            backgroundColor:
              recommendation === "durable"
                ? "rgba(6,78,59,0.08)"
                : "rgba(49,46,129,0.06)",
            color: "#9ca3af",
          }}
        >
          {recommendationText}
        </div>
      )}
    </div>
  );
};

export default PatternComparison;
