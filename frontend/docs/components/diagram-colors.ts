/**
 * Shared color constants for all documentation diagrams.
 *
 * Brand palette: sourced from the marketing CSS and docs global.css
 * State colors: match the dashboard badge conventions (badge.tsx / run-statuses.tsx)
 */

// ── Brand palette ──────────────────────────────────────────────────────────
export const brand = {
  navy: "#0A1029",
  navyDark: "#02081D",
  cyan: "#B8D9FF",
  cyanDark: "#A5C5E9",
  blue: "#3392FF",
  blueLight: "#85BDFF",
  magenta: "#BC46DD",
  magentaLight: "#D585EF",
  yellow: "#B8D41C",
} as const;

// ── State colors (dashboard-consistent) ────────────────────────────────────
export const state = {
  success: "#22C55E",
  successLight: "#4ADE80",
  successLighter: "#86EFAC",
  running: "#EAB308",
  runningLight: "#FACC15",
  runningDark: "#CA8A04",
  failed: "#EF4444",
  failedLight: "#FCA5A5",
  queued: "#64748B",
  cancelled: "#F97316",
} as const;

// ── Fills (used for node backgrounds at reduced opacity) ───────────────────
export const fill = {
  activeNode: "rgba(10, 16, 41, 0.3)",
  inactiveNode: "rgba(10, 16, 41, 0.15)",
  success: "rgba(34, 197, 94, 0.3)",
  successLight: "rgba(34, 197, 94, 0.1)",
  running: "rgba(234, 179, 8, 0.25)",
  runningLight: "rgba(234, 179, 8, 0.15)",
  failed: "rgba(239, 68, 68, 0.25)",
  magenta: "rgba(188, 70, 221, 0.2)",
  magentaLight: "rgba(188, 70, 221, 0.15)",
  blue: "rgba(51, 146, 255, 0.1)",
  dimmed: "rgba(10, 16, 41, 0.15)",
} as const;

// ── Container styling ──────────────────────────────────────────────────────
export const container = {
  border: "rgba(51, 146, 255, 0.2)",
  bg: "rgba(10, 16, 41, 0.04)",
} as const;

// ── Inactive / dimmed elements ─────────────────────────────────────────────
export const inactive = {
  stroke: "#1C2B4A",
  text: "#64748B",
  textLight: "#4A6080",
  fill: "#0A1029",
  edge: "#162035",
  line: "#1C2B4A",
  dot: "#162035",
  progress: "rgba(10, 16, 41, 0.5)",
} as const;

// ── Gradient stop pairs [start, end] for SVG linearGradient ────────────────
export const gradient = {
  blue: ["rgb(51, 146, 255)", "rgb(133, 189, 255)"] as const,
  magenta: ["rgb(188, 70, 221)", "rgb(213, 133, 239)"] as const,
  green: ["rgb(34, 197, 94)", "rgb(74, 222, 128)"] as const,
  yellow: ["rgb(234, 179, 8)", "rgb(250, 204, 21)"] as const,
  red: ["rgb(239, 68, 68)", "rgb(248, 113, 113)"] as const,
} as const;
