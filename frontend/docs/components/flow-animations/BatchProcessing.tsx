"use client";

import { useRef, type CSSProperties, type ReactNode } from "react";
import {
  Flow,
  defineTrack,
  useFlow,
  useFlowFrame,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import { Text } from "@/components/flow/Text";
import styles from "./batchprocessing.module.css";

/**
 * Flow animation of fan-out with limited concurrency
 * (docs.hatchet.run/v1/child-spawning): a stream of items queues up in a
 * single channel, and three processing slots pull from the head — so at most
 * three children run at once, no matter how fast items arrive. The queue
 * visibly backs up while all slots are busy, then drains as slots free.
 * Finished items fly to a completed grid on the right, where a tally counts
 * up to nine.
 *
 * The whole schedule is simulated once at module scope (greedy assignment of
 * queue head → first free slot), so the loop is fully deterministic and the
 * composition at t = DURATION matches the empty stage at t = 0.
 */

// ─── Geometry (stage design units, 320 × 200 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 200;

/** Source port and the single intake queue (head on the right). */
const PORT = { x: 12, y: 100 };
const QUEUE_Y = 100;
const HEAD_X = 124;
const PITCH = 12;
const CHANNEL_X0 = 36;
const CHANNEL_X1 = 132;
const RAIL_TOP = 91;
const RAIL_BOTTOM = 109;

const slotX = (slot: number) => HEAD_X - slot * PITCH;

/** Three processing slots, stacked — the concurrency limit made visible. */
const SLOTS = 3;
const WORKER_X = 176;
const WORKER_YS = [56, 100, 144] as const;

/** Completed grid: 3 × 3 cells, filled in completion order. */
const GRID_X0 = 246;
const GRID_Y0 = 87;
const GRID_PITCH = 13;
const gridPos = (k: number) => ({
  x: GRID_X0 + (k % 3) * GRID_PITCH,
  y: GRID_Y0 + Math.floor(k / 3) * GRID_PITCH,
});

const READOUT_X = 259;
const READOUT_Y = 130;

// ─── Schedule (simulated once, deterministically, at module scope) ─────────

const N_ITEMS = 9;
const SPAWN_GAP = 240;
const SPAWN0 = 150;
const SPAWNS = Array.from(
  { length: N_ITEMS },
  (_, i) => SPAWN0 + i * SPAWN_GAP,
);

const TRAVEL_IN = 500; // port → queue slot
const QUEUE_MIN = 200; // shortest stay at the head before dispatch
const MIN_HEADWAY = 380; // dispatches never overlap a queue shift
const TO_WORKER = 320; // head → slot
const WORKER_GAP = 160; // slot cooldown between children
const DONE_DWELL = 180; // green beat before flying to the grid
const FLY_MS = 420; // slot → completed grid
const SHIFT_MS = 300; // queue closes up behind each dispatch

/** Varied run durations — real children don't take identical time. */
const PROC_MS = [1400, 1750, 1500, 1650, 1400, 1800, 1550, 1450, 1600] as const;

interface Sched {
  /** When the item is dispatched from the queue head. */
  start: number;
  /** When its child finishes processing. */
  end: number;
  /** Which of the three slots ran it. */
  worker: number;
  /** When it lands in the completed grid. */
  landed: number;
  /** Its cell in the completed grid (completion order). */
  grid: number;
}

const SCHEDULE: Sched[] = (() => {
  const free = Array.from({ length: SLOTS }, () => 0);
  const rows: Omit<Sched, "landed" | "grid">[] = [];
  let lastStart = -Infinity;
  for (let i = 0; i < N_ITEMS; i++) {
    const ready = SPAWNS[i] + TRAVEL_IN + QUEUE_MIN;
    let worker = 0;
    for (let j = 1; j < SLOTS; j++) if (free[j] < free[worker]) worker = j;
    const start = Math.max(ready, free[worker], lastStart + MIN_HEADWAY);
    const end = start + TO_WORKER + PROC_MS[i];
    free[worker] = end + WORKER_GAP;
    lastStart = start;
    rows.push({ start, end, worker });
  }
  const byCompletion = rows
    .map((r, i) => ({ i, end: r.end }))
    .sort((a, b) => a.end - b.end);
  const grid = Array.from({ length: N_ITEMS }, () => 0);
  byCompletion.forEach(({ i }, rank) => {
    grid[i] = rank;
  });
  return rows.map((r, i) => ({
    ...r,
    landed: r.end + DONE_DWELL + FLY_MS,
    grid: grid[i],
  }));
})();

const ALL_LANDED = Math.max(...SCHEDULE.map((s) => s.landed));
const FADE_START = ALL_LANDED + 700; // hold the full grid so it reads
const FADE_END = FADE_START + 350;
const DURATION = FADE_END + 200;

/**
 * Static frame: all three slots busy, the queue backed up behind them, three
 * items already in the grid — the concurrency limit in one picture.
 */
const POSTER_TIME = 4300;

// ─── Track builders ────────────────────────────────────────────────────────

/**
 * One item: spawn at the port, join the queue at the current tail, shift
 * forward as earlier items dispatch, ride to a free slot (accent while its
 * child runs), then fly to its completed-grid cell and hold until the loop
 * fades out.
 */
const itemTrack = (i: number): FlowTrack => {
  const { start, end, worker, grid } = SCHEDULE[i];
  const spawn = SPAWNS[i];
  const arrive = spawn + TRAVEL_IN;
  const cell = gridPos(grid);
  const wy = WORKER_YS[worker];

  // Queue slot on arrival = my index minus everyone already dispatched.
  const startsBefore = SCHEDULE.filter(
    (s, k) => k < i && s.start <= arrive,
  ).length;
  let slot = i - startsBefore;

  const kfs: FlowKeyframe[] = [
    { t: spawn, x: PORT.x - 2, y: QUEUE_Y, opacity: 0, state: "queued" },
    { t: spawn + 150, x: 24, y: QUEUE_Y, opacity: 1, ease: "linear" },
    { t: spawn + 300, x: CHANNEL_X0 + 2, y: QUEUE_Y, ease: "linear" },
    { t: arrive, x: slotX(slot), y: QUEUE_Y, ease: "out" },
  ];
  // Close up one slot each time an earlier item dispatches after I arrive.
  for (let k = 0; k < i; k++) {
    const s = SCHEDULE[k].start;
    if (s > arrive) {
      kfs.push(
        { t: s, x: slotX(slot), y: QUEUE_Y, ease: "hold" },
        { t: s + SHIFT_MS, x: slotX(slot - 1), y: QUEUE_Y, ease: "inOut" },
      );
      slot -= 1;
    }
  }
  kfs.push(
    { t: start, x: HEAD_X, y: QUEUE_Y, ease: "hold" },
    {
      t: start + TO_WORKER,
      x: WORKER_X,
      y: wy,
      ease: "inOut",
      state: "processing",
    },
    { t: end, x: WORKER_X, y: wy, ease: "hold", state: "done" },
    { t: end + DONE_DWELL, x: WORKER_X, y: wy, ease: "hold" },
    {
      t: end + DONE_DWELL + FLY_MS,
      x: cell.x,
      y: cell.y,
      ease: "inOut",
      state: "completed",
    },
    { t: FADE_START, x: cell.x, y: cell.y, ease: "hold" },
    { t: FADE_END, x: cell.x, y: cell.y, opacity: 0, ease: "linear" },
  );
  return defineTrack(`bp-item-${i}`, kfs);
};

const ITEM_TRACKS = Array.from({ length: N_ITEMS }, (_, i) => itemTrack(i));

/** Each processing slot: busy for exactly one child at a time. */
const workerTrack = (j: number): FlowTrack => {
  const y = WORKER_YS[j];
  const kfs: FlowKeyframe[] = [{ t: 0, x: WORKER_X, y, state: "idle" }];
  for (const s of SCHEDULE) {
    if (s.worker !== j) continue;
    kfs.push(
      { t: s.start + TO_WORKER, x: WORKER_X, y, ease: "hold", state: "busy" },
      { t: s.end + DONE_DWELL, x: WORKER_X, y, ease: "hold", state: "idle" },
    );
  }
  kfs.push({ t: DURATION, x: WORKER_X, y, ease: "hold" });
  return defineTrack(`bp-worker-${j}`, kfs);
};

const WORKER_TRACKS = Array.from({ length: SLOTS }, (_, j) => workerTrack(j));

// ─── Completed tally (per-frame text, no React renders) ────────────────────

const clamp = (v: number, lo: number, hi: number) =>
  Math.min(hi, Math.max(lo, v));
const ramp = (t: number, a: number, b: number) =>
  clamp((t - a) / (b - a), 0, 1);

const completedCount = (t: number) =>
  SCHEDULE.filter((s) => t >= s.landed).length;

const readoutOpacity = (t: number) =>
  ramp(t, 900, 1150) * (1 - ramp(t, FADE_START, FADE_END));

/** The climbing tally under the grid: 0/9 → 9/9, one landing at a time. */
const CompletedReadout = () => {
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
    const next = String(completedCount(t));
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
        {completedCount(posterTime)}
      </span>
      /{N_ITEMS} done
    </div>
  );
};

// ─── Static chrome ─────────────────────────────────────────────────────────

const stroke = {
  fill: "none",
  strokeWidth: 1.5,
  vectorEffect: "non-scaling-stroke",
} as const;
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
    {/* Source port + approach into the queue */}
    <rect
      x={PORT.x - 2}
      y={PORT.y - 2}
      width={4}
      height={4}
      className={styles.chromeFill}
    />
    <line
      x1={18}
      y1={QUEUE_Y}
      x2={34}
      y2={QUEUE_Y}
      className={styles.chromeDash}
      {...fine}
    />
    {/* The intake queue — open tail, dispatched from the head */}
    <path
      d={`M ${CHANNEL_X0} ${RAIL_TOP + 5} L ${CHANNEL_X0} ${RAIL_TOP} L ${CHANNEL_X1} ${RAIL_TOP} L ${CHANNEL_X1} ${RAIL_TOP + 5}`}
      className={styles.chromeStroke}
      {...stroke}
    />
    <path
      d={`M ${CHANNEL_X0} ${RAIL_BOTTOM - 5} L ${CHANNEL_X0} ${RAIL_BOTTOM} L ${CHANNEL_X1} ${RAIL_BOTTOM} L ${CHANNEL_X1} ${RAIL_BOTTOM - 5}`}
      className={styles.chromeStroke}
      {...stroke}
    />
    {/* Head → the three slots */}
    {WORKER_YS.map((y) => (
      <line
        key={y}
        x1={CHANNEL_X1 + 2}
        y1={QUEUE_Y}
        x2={161}
        y2={y}
        className={styles.chromeDash}
        {...fine}
      />
    ))}
    {/* Slots → completed grid */}
    {WORKER_YS.map((y) => (
      <line
        key={y}
        x1={191}
        y1={y}
        x2={233}
        y2={QUEUE_Y}
        className={styles.chromeDash}
        {...fine}
      />
    ))}
    <line
      x1={233}
      y1={QUEUE_Y}
      x2={238}
      y2={QUEUE_Y}
      className={styles.chromeDash}
      {...fine}
    />
    {/* Completed grid outline */}
    <rect
      x={GRID_X0 - 7}
      y={GRID_Y0 - 7}
      width={GRID_PITCH * 2 + 14}
      height={GRID_PITCH * 2 + 14}
      className={styles.chromeGrid}
      {...fine}
      strokeDasharray="2 3"
    />
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
  "Animated diagram of fan-out with limited concurrency: a stream of nine items queues up in a single channel while three processing slots pull from the head, so at most three children run at once. The queue visibly backs up while all slots are busy, finished items fly to a completed grid, and a tally counts up from zero to nine.";

const CAPTION =
  "One child per item: three slots process in parallel while the rest of the queue waits.";

export const BatchProcessing = ({
  className,
  style,
  showCaption = true,
}: {
  className?: string;
  style?: CSSProperties;
  showCaption?: boolean;
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
        <StageLabel x={6} y={86} anchor="left">
          items
        </StageLabel>
        <StageLabel x={84} y={78}>
          queue
        </StageLabel>
        <StageLabel x={WORKER_X} y={30}>
          3 slots · concurrency
        </StageLabel>
        <StageLabel x={READOUT_X} y={66}>
          completed
        </StageLabel>
        {WORKER_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.workerBox} />
          </Flow.Token>
        ))}
        {ITEM_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.square} />
          </Flow.Token>
        ))}
        <CompletedReadout />
      </Flow.Stage>
    </Flow.Root>
    {showCaption && (
      <Text.Small as="p" secondary balance className={styles.caption}>
        {CAPTION}
      </Text.Small>
    )}
  </div>
);
