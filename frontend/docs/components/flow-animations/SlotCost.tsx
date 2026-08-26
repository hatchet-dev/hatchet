"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  defineTrack,
  resolveTrack,
  sampleTrack,
  type FlowKeyframe,
  type FlowTrack,
  type ResolvedFlowTrack,
} from "@/components/flow";
import { Text } from "@/components/flow/Text";
import styles from "./slotcost.module.css";

/**
 * Flow animation for task slot cost: one worker with a 4 × 5 grid of 20
 * slots, fed by two labelled streams. Cost-1 tasks are single squares that
 * fill one slot each and churn quickly; cost-5 tasks are bars of five stacked
 * squares — literally five slots tall — that dock onto a whole column and sit
 * there for most of the loop.
 *
 * The centrepiece beat is the wait. Mid-loop a cost-5 bar arrives while the
 * grid holds only a few scattered free slots — never five — so it parks at
 * the worker's edge, hollow and pulsing, while cost-1 squares keep slipping
 * past it into single free cells. Only when the running heavy task completes
 * and releases its five slots at once does the waiting bar move in. That is
 * the whole feature in one motion: every task draws from the same pool, a
 * heavy task just draws five at a time, and five scattered singles are not
 * the same thing as five together.
 *
 * The loop is a steady state, not a reset: tasks that outlive the loop
 * boundary (the heavy bars all do) are emitted as time-shifted slices of one
 * logical track, so the nearly-full grid at t = 0 is the tail of the
 * previous iteration. All scheduling is plain data at module scope.
 */

// ─── Geometry (stage design units, 320 × 200 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 200;

/** Source ports: heavy tasks on the upper lane, light tasks on the lower. */
const PORT_X = 6;
const OMEGA_Y = 90;
const WEENIE_Y = 142;

/** Admission gate before the worker; heavy tasks that can't fit park here. */
const GATE_X = 186;
const WAIT_X = 170;

/** The slot grid: 4 columns × 5 rows, one cell per slot. */
const CELL = 13;
const COL_X = [232, 249, 266, 283] as const;
const ROW_Y = [82, 99, 116, 133, 150] as const;
const GRID_CY = ROW_Y[2];

/** Worker chassis around the grid. */
const BOX = { x: 216, y: 66, w: 84, h: 100 } as const;

// ─── Timing (ms) ───────────────────────────────────────────────────────────

/**
 * The whole schedule below is authored on a 16s score; TEMPO stretches it
 * uniformly at emit time (loopTracks), so every dock/dwell/travel figure in
 * this file is score time — multiply by TEMPO for real ms.
 */
const SCORE_DURATION = 16000;
const TEMPO = 1.25;
const DURATION = SCORE_DURATION * TEMPO;

/**
 * Static frame: two heavy bars docked (columns 0 and 1), eight light tasks
 * spread over the two right columns, two more inbound on the lower lane —
 * 18 of 20 slots in use. The capacity mix is legible without motion.
 */
const POSTER_TIME = 1500 * TEMPO;

/** Shared travel profile: fade-in hop, lane run, then the hop into a slot. */
const W_LEAD = 1150; // weenie spawn → dock
const W_FADE_IN = 160;
const W_HOP = 350; // gate → cell
const W_DONE = 260; // completion fade
const O_LEAD = 1340; // omega spawn → dock (heavier, slower)
const O_HOP = 340;
const O_DONE = 300;

// ─── Loop wrapping ─────────────────────────────────────────────────────────

/**
 * Heavy tasks outlive the loop and the grid is pre-filled at t = 0 by the
 * previous iteration's arrivals. Same trick as FairQueueing: build each track
 * once in an unbounded time domain, then emit the slices that intersect
 * [0, DURATION] under every whole-loop shift, sampling the boundary values
 * from the track itself so t = DURATION matches t = 0 exactly. Segments that
 * can straddle a boundary are linear or hold.
 */
const SHIFTS = [-2, -1, 0, 1].map((n) => n * DURATION);

const boundaryKeyframe = (resolved: ResolvedFlowTrack, t: number): FlowKeyframe => {
  const s = sampleTrack(resolved, t);
  return { t, x: s.x, y: s.y, opacity: s.opacity, scale: s.scale, state: s.state, ease: "linear" };
};

const clipToLoop = (id: string, keyframes: FlowKeyframe[]): FlowTrack | null => {
  const first = keyframes[0].t;
  const last = keyframes[keyframes.length - 1].t;
  if (last <= 0 || first >= DURATION) return null;
  if (first >= 0 && last <= DURATION) return defineTrack(id, keyframes);

  const resolved = resolveTrack({ id, keyframes });
  const clipped: FlowKeyframe[] = [];
  if (first < 0) clipped.push(boundaryKeyframe(resolved, 0));
  for (const k of keyframes) {
    if (k.t < 0 || k.t > DURATION) continue;
    if (first < 0 && k.t === 0) continue;
    if (last > DURATION && k.t === DURATION) continue;
    clipped.push(k);
  }
  if (last > DURATION) clipped.push(boundaryKeyframe(resolved, DURATION));
  return clipped.length > 1 ? defineTrack(id, clipped) : null;
};

/** Every visible slice of one logically unbounded track (score → real ms). */
const loopTracks = (id: string, keyframes: FlowKeyframe[]): FlowTrack[] =>
  SHIFTS.map((shift) =>
    clipToLoop(
      `${id}~${shift}`,
      keyframes.map((k) => ({ ...k, t: k.t * TEMPO + shift }))
    )
  ).filter((track): track is FlowTrack => track !== null);

// ─── Light tasks (cost 1): per-cell occupancy schedules ────────────────────

/**
 * Each entry is one slot cell and the [dock, complete) intervals of the
 * cost-1 tasks that pass through it — per-cell lists, so double-booking a
 * slot is impossible by construction, and every interval repeats each loop.
 * Column 0 is omega territory all loop; column 1 hosts light tasks only in
 * the window between the heavy task leaving at 6800 and the next docking at
 * 14000. The schedule is tuned so that while the third heavy bar waits at
 * the gate (8500 → 10600) the grid never has five slots free at once.
 */
interface CellRuns {
  col: number;
  row: number;
  runs: ReadonlyArray<readonly [number, number]>;
}

const CELL_SCHEDULE: readonly CellRuns[] = [
  { col: 2, row: 0, runs: [[300, 3600], [4600, 7900], [8300, 11800], [12700, 15400]] },
  { col: 2, row: 1, runs: [[15600, 18900], [3800, 7000], [8100, 11200], [12100, 14800]] },
  { col: 2, row: 2, runs: [[900, 4200], [5000, 8100], [8600, 11900], [12800, 15700]] },
  { col: 2, row: 3, runs: [[1900, 5300], [6400, 9800], [10900, 14300], [15200, 17300]] },
  { col: 2, row: 4, runs: [[14600, 17800], [2700, 6200], [7100, 10600], [11500, 13900]] },
  { col: 3, row: 0, runs: [[500, 3900], [4800, 8300], [9100, 12300], [13200, 15600]] },
  { col: 3, row: 1, runs: [[15900, 19200], [4100, 7600], [8400, 11700], [12500, 15100]] },
  { col: 3, row: 2, runs: [[1200, 4700], [5600, 9000], [9700, 13100], [14200, 16700]] },
  { col: 3, row: 3, runs: [[2300, 5800], [6700, 10200], [11100, 14600], [15500, 17800]] },
  { col: 3, row: 4, runs: [[400, 3000], [3900, 7400], [8000, 11400], [12200, 15300]] },
  { col: 1, row: 0, runs: [[7600, 10500], [11300, 13200]] },
  { col: 1, row: 1, runs: [[8200, 11600]] },
  { col: 1, row: 2, runs: [[7200, 9700], [10400, 12900]] },
  { col: 1, row: 3, runs: [[8800, 12300]] },
  { col: 1, row: 4, runs: [[7900, 10100], [11000, 13100]] },
];

const weenieKeyframes = (dock: number, end: number, cx: number, ry: number): FlowKeyframe[] => {
  const spawn = dock - W_LEAD;
  return [
    { t: spawn, x: PORT_X, y: WEENIE_Y, opacity: 0, state: "inflight" },
    { t: spawn + W_FADE_IN, x: PORT_X + 18, y: WEENIE_Y, opacity: 1, ease: "linear" },
    { t: dock - W_HOP, x: GATE_X - 6, y: WEENIE_Y, ease: "linear" },
    { t: dock, x: cx, y: ry, ease: "out", state: "running" },
    { t: end, x: cx, y: ry, ease: "hold", state: "done" },
    { t: end + W_DONE, x: cx, y: ry, opacity: 0, scale: 0.5, ease: "linear" },
  ];
};

const WEENIE_TRACKS = CELL_SCHEDULE.flatMap(({ col, row, runs }) =>
  runs.flatMap(([dock, end], i) =>
    loopTracks(`sc-w${col}${row}-${i}`, weenieKeyframes(dock, end, COL_X[col], ROW_Y[row]))
  )
);

// ─── Heavy tasks (cost 5): one bar per column reservation ──────────────────

/**
 * Three heavy tasks per loop. The first two dock as soon as they arrive; the
 * third reaches the gate at 8500 with the grid too fragmented to hold it —
 * at most four scattered slots free — and waits until the column-0 task
 * completes at 10300, releasing five slots at once. It then runs across the
 * loop boundary and is the bar already docked in column 0 at t = 0.
 */
interface OmegaSpec {
  id: string;
  col: number;
  dock: number;
  end: number;
  /** Arrive at the gate and hold until admitted at `dock - O_HOP`. */
  waitFrom?: number;
}

// Render order is stacking order: the waiting task is listed last so that
// when the column-1 arrival hops across the docked column-0 bar it passes
// behind it, not through it.
const OMEGAS: readonly OmegaSpec[] = [
  { id: "sc-o-mid", col: 0, dock: 3900, end: 10300 },
  { id: "sc-o-right", col: 1, dock: 14000, end: 22800 },
  { id: "sc-o-wait", col: 0, dock: 10940, end: 18900, waitFrom: 8500 },
];

const omegaKeyframes = ({ col, dock, end, waitFrom }: OmegaSpec): FlowKeyframe[] => {
  const gateAt = waitFrom ?? dock - O_HOP;
  const spawn = gateAt - (O_LEAD - O_HOP);
  const keyframes: FlowKeyframe[] = [
    { t: spawn, x: PORT_X, y: OMEGA_Y, opacity: 0, state: "inflight" },
    { t: spawn + W_FADE_IN, x: PORT_X + 18, y: OMEGA_Y, opacity: 1, ease: "linear" },
    { t: gateAt, x: WAIT_X, y: OMEGA_Y, ease: "linear", state: waitFrom ? "waiting" : "inflight" },
  ];
  // The admission moment: the bar fills the instant five slots are granted,
  // then slides in solid.
  if (waitFrom) keyframes.push({ t: dock - O_HOP, x: WAIT_X, y: OMEGA_Y, ease: "hold", state: "running" });
  keyframes.push(
    { t: dock, x: COL_X[col], y: GRID_CY, ease: "out", state: "running" },
    { t: end, x: COL_X[col], y: GRID_CY, ease: "hold", state: "done" },
    { t: end + O_DONE, x: COL_X[col], y: GRID_CY, opacity: 0, scale: 0.6, ease: "linear" }
  );
  return keyframes;
};

const OMEGA_TRACKS = OMEGAS.flatMap((spec) => loopTracks(spec.id, omegaKeyframes(spec)));

// ─── Static chrome ─────────────────────────────────────────────────────────

const stroke = { fill: "none", strokeWidth: 1.5, vectorEffect: "non-scaling-stroke" } as const;
const fine = { fill: "none", strokeWidth: 1, vectorEffect: "non-scaling-stroke" } as const;

const Chrome = () => (
  <svg viewBox={`0 0 ${STAGE_W} ${STAGE_H}`} aria-hidden="true" className={styles.chrome}>
    {/* Source ports + the two inbound lanes */}
    {[OMEGA_Y, WEENIE_Y].map((y) => (
      <g key={y}>
        <rect x={PORT_X - 2} y={y - 2} width={4} height={4} className={styles.chromeFill} />
        <line x1={PORT_X + 4} y1={y} x2={GATE_X - 10} y2={y} className={styles.chromeDash} {...fine} />
      </g>
    ))}
    {/* Admission gate */}
    <line x1={GATE_X} y1={62} x2={GATE_X} y2={170} className={styles.chromeRail} {...fine} />
    {/* Worker chassis */}
    <rect x={BOX.x} y={BOX.y} width={BOX.w} height={BOX.h} className={styles.chromeStroke} {...stroke} />
    {/* One outlined cell per slot */}
    {COL_X.map((cx) =>
      ROW_Y.map((ry) => (
        <rect
          key={`${cx}-${ry}`}
          x={cx - CELL / 2}
          y={ry - CELL / 2}
          width={CELL}
          height={CELL}
          className={styles.chromeCell}
          {...fine}
        />
      ))
    )}
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
    className={[styles.stageLabel, anchor === "left" ? styles.anchorLeft : "", muted ? styles.labelMuted : ""]
      .filter(Boolean)
      .join(" ")}
    style={{ left: `calc(var(--flow-u) * ${x})`, top: `calc(var(--flow-u) * ${y})` }}
  >
    {children}
  </div>
);

// ─── Export ────────────────────────────────────────────────────────────────

const ARIA_LABEL =
  "Animated diagram of task slot cost: a worker with a grid of 20 slots runs a mix of tasks. Cost-1 tasks are single squares that each fill one slot and finish quickly. Cost-5 tasks are bars of five stacked squares that reserve a whole column of five slots for much longer. Mid-loop a cost-5 bar arrives while only a few scattered slots are free, so it waits at the worker's edge while cost-1 squares keep slipping into single free slots; when a running cost-5 task completes and frees five slots at once, the waiting bar moves in.";

const CAPTION =
  "Every task draws from the same pool of worker slots — a cost-5 task reserves five at once, and waits until five are free.";

export const SlotCost = ({
  style,
  showCaption = true,
}: {
  style?: CSSProperties;
  showCaption?: boolean;
}) => (
  <div className={styles.wrap} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.flowRoot}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={PORT_X + 2} y={36} anchor="left">
          cost 5
        </StageLabel>
        <StageLabel x={PORT_X + 2} y={155} anchor="left">
          cost 1
        </StageLabel>
        <StageLabel x={BOX.x + BOX.w / 2} y={50} muted>
          worker · 20 slots
        </StageLabel>
        {WEENIE_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.weenie} />
          </Flow.Token>
        ))}
        {OMEGA_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.bar}>
              {Array.from({ length: 5 }, (_, i) => (
                <div key={i} className={styles.seg} />
              ))}
            </div>
          </Flow.Token>
        ))}
      </Flow.Stage>
    </Flow.Root>
    {showCaption && (
      <Text.Small as="p" secondary balance className={styles.caption}>
        {CAPTION}
      </Text.Small>
    )}
  </div>
);
