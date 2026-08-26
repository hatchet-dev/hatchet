"use client";

import { useRef, type CSSProperties, type ReactNode } from "react";
import { Flow, defineTrack, useFlow, useFlowFrame } from "@/components/flow";
import styles from "./childfanout.module.css";

/**
 * Flow animation of Hatchet child spawning (docs.hatchet.run/v1/child-spawning):
 * a parent run spawns N children that fan out to parallel lanes and run
 * concurrently. Each child finishes at its own pace — deliberately out of
 * spawn order, because they really are parallel — flips to a terminal status,
 * and its result travels back down the same lane to the parent. The parent
 * holds an awaiting state (accent border) until the last result lands, then
 * flips complete. A small readout under the parent tallies collected results.
 *
 * All timings are hand-scheduled at module scope, so the loop is fully
 * deterministic and the composition at t = DURATION matches t = 0.
 */

// ─── Geometry (stage design units, 320 × 208 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 208;

/** The parent run box on the left. */
const PARENT_X = 52;
const PARENT_Y = 104;
const PARENT_EDGE = 74; // right edge of the parent box (box is 44 wide)

/** Child lanes: four parallel slots on the right. */
const CHILD_X = 196;
const LANE_YS = [44, 84, 124, 164] as const;
const LANE_LABELS = ["child 1", "child 2", "child 3", "child n"] as const;

const READOUT_X = PARENT_X;
const READOUT_Y = 128;

// ─── Timing (ms) ───────────────────────────────────────────────────────────

/** Children leave the parent staggered — one spawn call per item. */
const SPAWNS = [500, 670, 840, 1010] as const;
const OUT_MS = 650; // parent → lane slot

/**
 * Finish times are hand-picked and out of spawn order: child 2 finishes
 * first, child 1 third — parallel work completes when it completes.
 */
const FINISHES = [4300, 3000, 4900, 3700] as const;
const DONE_DWELL = 350; // terminal-status beat before the result departs
const BACK_MS = 550; // lane slot → parent
const ABSORB_MS = 200; // result fades into the parent

/** When each child's result lands back at the parent. */
const ARRIVALS = FINISHES.map((f) => f + DONE_DWELL + BACK_MS);
const LAST_ARRIVAL = Math.max(...ARRIVALS); // 5800

const COMPLETE_HOLD = 7000; // parent settles back to idle here
const DURATION = 7600;

/**
 * Static frame: three children still running in parallel, one result already
 * collected and another on its way back — the whole fan-out/fan-in story.
 */
const POSTER_TIME = 4200;

// ─── Tracks ────────────────────────────────────────────────────────────────

/**
 * One child: bud off the parent's edge, fan out to its lane, run (accent),
 * reach terminal status (green), then carry the result back and fade into
 * the parent.
 */
const childTrack = (i: number) => {
  const spawn = SPAWNS[i];
  const finish = FINISHES[i];
  const laneY = LANE_YS[i];
  const depart = finish + DONE_DWELL;
  return defineTrack(`cf-child-${i}`, [
    {
      t: spawn,
      x: PARENT_EDGE,
      y: PARENT_Y,
      opacity: 0,
      scale: 0.7,
      state: "spawned",
    },
    {
      t: spawn + 150,
      x: 90,
      y: PARENT_Y + (laneY - PARENT_Y) * 0.15,
      opacity: 1,
      scale: 1,
      ease: "linear",
    },
    { t: spawn + OUT_MS, x: CHILD_X, y: laneY, ease: "out", state: "running" },
    { t: finish, x: CHILD_X, y: laneY, ease: "hold", state: "done" },
    { t: depart, x: CHILD_X, y: laneY, ease: "hold", state: "result" },
    { t: depart + BACK_MS, x: 86, y: PARENT_Y, ease: "inOut" },
    {
      t: depart + BACK_MS + ABSORB_MS,
      x: PARENT_EDGE,
      y: PARENT_Y,
      opacity: 0,
      scale: 0.6,
      ease: "in",
    },
  ]);
};

const CHILD_TRACKS = LANE_YS.map((_, i) => childTrack(i));

/** The parent run: awaiting while any child is out, complete when all return. */
const PARENT_TRACK = defineTrack("cf-parent", [
  { t: 0, x: PARENT_X, y: PARENT_Y, state: "idle" },
  { t: SPAWNS[0], x: PARENT_X, y: PARENT_Y, ease: "hold", state: "awaiting" },
  {
    t: LAST_ARRIVAL,
    x: PARENT_X,
    y: PARENT_Y,
    ease: "hold",
    state: "complete",
  },
  { t: COMPLETE_HOLD, x: PARENT_X, y: PARENT_Y, ease: "hold", state: "idle" },
  { t: DURATION, x: PARENT_X, y: PARENT_Y, ease: "hold" },
]);

// ─── Results readout (per-frame text, no React renders) ────────────────────

const clamp = (v: number, lo: number, hi: number) =>
  Math.min(hi, Math.max(lo, v));
const ramp = (t: number, a: number, b: number) =>
  clamp((t - a) / (b - a), 0, 1);

const resultsCollected = (t: number) => ARRIVALS.filter((a) => t >= a).length;

const readoutOpacity = (t: number) =>
  ramp(t, 1100, 1350) * (1 - ramp(t, 6800, 7100));

/** The await-all tally: fills in as results land, out of spawn order. */
const ResultsReadout = () => {
  const { posterTime } = useFlow();
  const rootRef = useRef<HTMLDivElement | null>(null);
  const countRef = useRef<HTMLSpanElement | null>(null);

  useFlowFrame((t) => {
    const root = rootRef.current;
    if (!root) return;
    const opacity = readoutOpacity(t);
    root.style.opacity = String(opacity);
    root.style.visibility = opacity <= 0.001 ? "hidden" : "";
    const count = countRef.current;
    const next = String(resultsCollected(t));
    if (count && count.textContent !== next) count.textContent = next;
  });

  const initialOpacity = readoutOpacity(posterTime);

  return (
    <div
      ref={rootRef}
      className={styles.readout}
      style={{
        left: `calc(var(--flow-u) * ${READOUT_X})`,
        top: `calc(var(--flow-u) * ${READOUT_Y})`,
        opacity: initialOpacity,
        visibility: initialOpacity <= 0.001 ? "hidden" : undefined,
      }}
    >
      <span ref={countRef} className={styles.readoutCount}>
        {resultsCollected(posterTime)}
      </span>
      /4 results
    </div>
  );
};

// ─── Static chrome ─────────────────────────────────────────────────────────

const fine = {
  fill: "none",
  strokeWidth: 1,
  vectorEffect: "non-scaling-stroke",
} as const;

const Chrome = () => (
  <svg
    viewBox={`0 0 ${STAGE_W} ${STAGE_H}`}
    aria-hidden="true"
    className={styles.chrome}
  >
    {/* Fan lines: one spawn path per child, out and back on the same lane */}
    {LANE_YS.map((y) => (
      <line
        key={y}
        x1={PARENT_EDGE + 2}
        y1={PARENT_Y}
        x2={186}
        y2={y}
        className={styles.chromeDash}
        {...fine}
      />
    ))}
    {/* Child run slots */}
    {LANE_YS.map((y) => (
      <rect
        key={y}
        x={CHILD_X - 8}
        y={y - 8}
        width={16}
        height={16}
        className={styles.chromeSlot}
        {...fine}
        strokeDasharray="2 3"
      />
    ))}
  </svg>
);

const StageLabel = ({
  x,
  y,
  anchor = "center",
  muted = false,
  children,
}: {
  x: number;
  y: number;
  anchor?: "center" | "left";
  muted?: boolean;
  children: ReactNode;
}) => (
  <div
    className={`${styles.stageLabel} ${anchor === "left" ? styles.anchorLeft : ""} ${
      muted ? styles.labelMuted : ""
    }`}
    style={{
      left: `calc(var(--flow-u) * ${x})`,
      top: `calc(var(--flow-u) * ${y})`,
    }}
  >
    {children}
  </div>
);

// ─── Export ────────────────────────────────────────────────────────────────

const ARIA_LABEL =
  "Animated diagram of Hatchet child spawning: a parent run spawns four children that fan out to parallel lanes and run concurrently. Each child finishes at its own pace, flips to a completed status, and returns its result to the parent. The parent waits with an awaiting state until the last result lands, then completes.";

export const ChildFanout = ({
  className,
  style,
}: {
  className?: string;
  style?: CSSProperties;
}) => (
  <div className={`${styles.wrap} ${className ?? ""}`} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.flowRoot}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={PARENT_X} y={80}>
          parent
        </StageLabel>
        {LANE_YS.map((y, i) => (
          <StageLabel key={y} x={210} y={y - 2.5} anchor="left" muted>
            {LANE_LABELS[i]}
          </StageLabel>
        ))}
        <Flow.Token track={PARENT_TRACK}>
          <div className={styles.parentBox} />
        </Flow.Token>
        {CHILD_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.square} />
          </Flow.Token>
        ))}
        <ResultsReadout />
      </Flow.Stage>
    </Flow.Root>
  </div>
);
