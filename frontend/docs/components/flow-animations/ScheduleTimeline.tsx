"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  createSeededRandom,
  defineTrack,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import { Text } from "@/components/flow/Text";
import styles from "./scheduletimeline.module.css";

/**
 * Flow animation of reliable future scheduling
 * (docs.hatchet.run/v1/scheduled-runs).
 *
 * A future timeline stretches right; a NOW cursor sweeps along it. Each
 * schedule is posted *at* the cursor — the run token pops out of NOW and lobs
 * forward onto its trigger time, where a dashed durable record materializes
 * around it. Posted schedules then sit completely still (durability is the
 * absence of motion) until the cursor reaches them: the record flashes, the
 * run drops onto the dispatch lane, processes at the worker in accent blue,
 * and settles into a small run-history strip. One schedule is posted seconds
 * into the loop but fires near its end — post now, runs much later,
 * guaranteed.
 */

// ─── Geometry (stage design units, 320 × 204 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 204;

const AXIS_Y = 70; // the timeline itself
const MARKER_Y = 50; // posted schedule entries hover above the line
const LANE_Y = 130; // dispatch lane fired runs drop onto
const WORKER_X = 264;

/** Ticks: fine every 14 units, taller majors every 56 (three labelled). */
const TICK_X0 = 26;
const TICK_PITCH = 14;
const TICK_COUNT = 20;
const MAJOR_EVERY = 4;
const MAJORS_LABELLED: [number, string][] = [
  [40, "12:00"],
  [152, "12:30"],
  [264, "13:00"],
];

/** NOW cursor sweep: fades in at the left, sweeps the visible future, fades. */
const PLAY_X0 = 36;
const PLAY_X1 = 296;
const SWEEP_T0 = 600;
const SWEEP_MS = 15000;

const playX = (t: number) =>
  PLAY_X0 + ((t - SWEEP_T0) / SWEEP_MS) * (PLAY_X1 - PLAY_X0);
const fireAt = (x: number) =>
  SWEEP_T0 + ((x - PLAY_X0) / (PLAY_X1 - PLAY_X0)) * SWEEP_MS;

/** Completed-run history strip (bottom left). */
const REGION = { x: 16, y: 152, w: 174, h: 40 };
const CELL = 6.5;
const CELL_PITCH = 9.5;
const CELL_COLS = 17;
const CELL_X0 = 26;
const CELL_ROWS: { y: number; fillP: number }[] = [
  { y: 169, fillP: 0.6 },
  { y: 179, fillP: 0.34 },
];

// ─── Pre-seeded run history (dim texture; live runs land in the gaps) ──────

interface HistCell {
  x: number; // cell center, stage units
  y: number;
}

const HISTORY = (() => {
  const histRng = createSeededRandom("schedule-history");
  const fg: [string, string, string] = ["", "", ""];
  let blue = "";
  let green = "";
  const gaps: HistCell[] = [];
  for (const row of CELL_ROWS) {
    for (let c = 0; c < CELL_COLS; c++) {
      const x = CELL_X0 + c * CELL_PITCH;
      if (histRng() >= row.fillP) {
        gaps.push({ x: x + CELL / 2, y: row.y + CELL / 2 });
        continue;
      }
      const cell = `M${x} ${row.y}h${CELL}v${CELL}h-${CELL}z`;
      const tint = histRng();
      if (tint < 0.05) {
        blue += cell;
      } else if (tint < 0.09) {
        green += cell;
      } else {
        const b = histRng();
        fg[b < 0.32 ? 0 : b < 0.72 ? 1 : 2] += cell;
      }
    }
  }
  return { fg, blue, green, gaps };
})();

const HIST_OPACITIES = [0.3, 0.48, 0.68] as const;
/** Landed runs grow slightly to match the history cell size (payload is 5u). */
const LAND_SCALE = CELL / 5;

// ─── Schedule sim (all times derived from the cursor sweep) ────────────────

const POST_FLIGHT = 650; // API pop → landing on the trigger time
const FIRE_BEAT = 140; // dwell at the marker the instant NOW arrives
const DROP_MS = 420; // marker → dispatch lane
const SLIDE_PER_UNIT = 5; // lane → worker, distance-proportional
const DONE_DWELL = 350;
const FLY_MS = 480; // worker → history cell

interface Sched {
  x: number; // trigger-time position on the timeline
  post: number; // when it's created via the API
  origin: number; // cursor position at post time (schedules are posted at NOW)
  land: number;
  fire: number;
  arrive: number;
  procEnd: number;
  storeAt: number;
  cell: HistCell;
}

const timingRng = createSeededRandom("schedule-timing");

/**
 * Three schedules per loop. The second is the hero: posted in the loop's
 * opening seconds at the far end of the timeline, it waits almost the entire
 * loop before firing.
 */
const SCHEDULES: Sched[] = [
  { x: 124, post: 900 },
  { x: 234, post: 1800 },
  { x: 178, post: 4600 },
].map((s, i) => {
  const origin = Math.max(PLAY_X0, playX(s.post));
  const fire = Math.round(fireAt(s.x));
  const arrive =
    fire + FIRE_BEAT + DROP_MS + Math.max(260, (WORKER_X - s.x) * SLIDE_PER_UNIT);
  const procEnd = arrive + Math.round(1100 + timingRng() * 300);
  const gapIdx = Math.round((i * (HISTORY.gaps.length - 1)) / 2);
  return {
    ...s,
    origin,
    fire,
    land: s.post + POST_FLIGHT,
    arrive,
    procEnd,
    storeAt: procEnd + DONE_DWELL + FLY_MS,
    cell: HISTORY.gaps[gapIdx],
  };
});

// ─── Loop bookkeeping ──────────────────────────────────────────────────────

const lastStore = Math.max(...SCHEDULES.map((s) => s.storeAt));
/** Everything (cursor, records, fresh cells) fades in the final beat, so t = DURATION matches t = 0. */
const DURATION = Math.ceil((lastStore + 1900) / 100) * 100;
/**
 * Static frame: the cursor is mid-sweep, the first run is processing in
 * accent blue, and two durable schedules wait ahead of NOW.
 */
const POSTER_TIME = Math.round((SCHEDULES[0].arrive + SCHEDULES[0].procEnd) / 2);

// ─── Tracks ────────────────────────────────────────────────────────────────

/** The run token: posted at NOW, lobbed onto its trigger time, then patient. */
const payloadTrack = (s: Sched, i: number): FlowTrack => {
  const settleAt = Math.min(s.storeAt + 1500, DURATION - 360);
  return defineTrack(`payload-${i}`, [
    { t: s.post, x: s.origin, y: 62, opacity: 0, scale: 0.6, state: "posting" },
    { t: s.post + 140, x: s.origin + 6, y: 54, opacity: 1, scale: 1, ease: "linear" },
    { t: s.post + 400, x: (s.origin + s.x) / 2, y: 28, ease: "linear" },
    { t: s.land, x: s.x, y: MARKER_Y, ease: "out", state: "scheduled" },
    { t: s.fire, x: s.x, y: MARKER_Y, ease: "hold", state: "fired" },
    { t: s.fire + FIRE_BEAT, x: s.x, y: MARKER_Y, ease: "hold" },
    { t: s.fire + FIRE_BEAT + DROP_MS, x: s.x, y: LANE_Y, ease: "in" },
    { t: s.arrive, x: WORKER_X, y: LANE_Y, ease: "inOut", state: "active" },
    { t: s.procEnd, x: WORKER_X, y: LANE_Y, ease: "hold", state: "done" },
    { t: s.procEnd + DONE_DWELL, x: WORKER_X, y: LANE_Y, ease: "hold" },
    { t: s.storeAt, x: s.cell.x, y: s.cell.y, scale: LAND_SCALE, ease: "inOut", state: "stored" },
    { t: settleAt, x: s.cell.x, y: s.cell.y, scale: LAND_SCALE, ease: "hold", state: "settled" },
    { t: DURATION - 320, x: s.cell.x, y: s.cell.y, scale: LAND_SCALE, ease: "hold" },
    { t: DURATION, x: s.cell.x, y: s.cell.y, scale: LAND_SCALE, opacity: 0, ease: "linear" },
  ]);
};

/** The dashed durable record that materializes around a posted schedule. */
const recordTrack = (s: Sched, i: number): FlowTrack =>
  defineTrack(`record-${i}`, [
    { t: s.land - 40, x: s.x, y: MARKER_Y, opacity: 0, scale: 0.5, state: "record" },
    { t: s.land + 60, x: s.x, y: MARKER_Y, opacity: 1, scale: 1.12, ease: "out" },
    { t: s.land + 240, x: s.x, y: MARKER_Y, scale: 1, ease: "inOut" },
    { t: s.fire, x: s.x, y: MARKER_Y, ease: "hold", state: "firing" },
    { t: s.fire + 700, x: s.x, y: MARKER_Y, ease: "hold" },
    { t: s.fire + 1000, x: s.x, y: MARKER_Y, opacity: 0, ease: "linear" },
  ]);

/** The small fired-at tick a consumed schedule leaves on the timeline. */
const residueTrack = (s: Sched, i: number): FlowTrack =>
  defineTrack(`residue-${i}`, [
    { t: s.fire + 250, x: s.x, y: AXIS_Y, opacity: 0, state: "residue" },
    { t: s.fire + 550, x: s.x, y: AXIS_Y, opacity: 1, ease: "linear" },
    { t: DURATION - 500, x: s.x, y: AXIS_Y, ease: "hold" },
    { t: DURATION, x: s.x, y: AXIS_Y, opacity: 0, ease: "linear" },
  ]);

const PAYLOADS = SCHEDULES.map(payloadTrack);
const RECORDS = SCHEDULES.map(recordTrack);
const RESIDUES = SCHEDULES.map(residueTrack);

/** NOW cursor: fade in at the left edge, sweep the loop, fade at the far end. */
const cursorKeyframes = (y: number): FlowKeyframe[] => [
  { t: 0, x: PLAY_X0, y, opacity: 0 },
  { t: 200, x: PLAY_X0, y, opacity: 0, ease: "hold" },
  { t: SWEEP_T0, x: PLAY_X0, y, opacity: 1, ease: "linear" },
  { t: SWEEP_T0 + SWEEP_MS, x: PLAY_X1, y, ease: "linear" },
  { t: SWEEP_T0 + SWEEP_MS + 500, x: PLAY_X1, y, opacity: 0, ease: "linear" },
];

const CURSOR_LINE = defineTrack("cursor-line", cursorKeyframes(57));
const CURSOR_TAG = defineTrack("cursor-tag", cursorKeyframes(26));

const WORKER = defineTrack("worker", [
  { t: 0, x: WORKER_X, y: LANE_Y, state: "idle" },
  // Schedules are declared in post order; the worker sees them in fire order.
  ...[...SCHEDULES].sort((a, b) => a.arrive - b.arrive).flatMap((s): FlowKeyframe[] => [
    { t: s.arrive, x: WORKER_X, y: LANE_Y, state: "busy", ease: "hold" },
    { t: s.procEnd, x: WORKER_X, y: LANE_Y, state: "idle", ease: "hold" },
  ]),
  { t: DURATION, x: WORKER_X, y: LANE_Y, ease: "hold" },
]);

// ─── Static chrome ─────────────────────────────────────────────────────────

const stroke = { fill: "none", strokeWidth: 1.5, vectorEffect: "non-scaling-stroke" } as const;
const fine = { fill: "none", strokeWidth: 1, vectorEffect: "non-scaling-stroke" } as const;

const Chrome = () => (
  <svg viewBox={`0 0 ${STAGE_W} ${STAGE_H}`} aria-hidden="true">
    {/* Timeline axis, running off toward the future */}
    <rect x={12.5} y={AXIS_Y - 1.5} width={3} height={3} className={styles.chromeFill} />
    <line x1={14} y1={AXIS_Y} x2={299} y2={AXIS_Y} className={styles.chromeStroke} {...stroke} />
    <path
      d={`M 296.5 ${AXIS_Y - 4} L 302 ${AXIS_Y} L 296.5 ${AXIS_Y + 4}`}
      className={styles.chromeStroke}
      {...stroke}
    />
    {/* Fine ticks, taller majors */}
    {Array.from({ length: TICK_COUNT }, (_, k) => {
      const x = TICK_X0 + k * TICK_PITCH;
      const major = k % MAJOR_EVERY === 1;
      const h = major ? 3.5 : 2;
      return (
        <line
          key={k}
          x1={x}
          y1={AXIS_Y - h}
          x2={x}
          y2={AXIS_Y + h}
          className={major ? styles.chromeMajorTick : styles.chromeTick}
          {...fine}
        />
      );
    })}
    {/* Dispatch lane toward the worker */}
    <line x1={112} y1={LANE_Y} x2={238} y2={LANE_Y} className={styles.chromeDash} {...fine} />
    {/* Run history strip: dashed frame + corner marks */}
    <rect
      x={REGION.x}
      y={REGION.y}
      width={REGION.w}
      height={REGION.h}
      className={styles.chromeDashRect}
      {...fine}
      strokeDasharray="2 3"
    />
    {[
      [REGION.x, REGION.y],
      [REGION.x + REGION.w, REGION.y],
      [REGION.x, REGION.y + REGION.h],
      [REGION.x + REGION.w, REGION.y + REGION.h],
    ].map(([cx, cy]) => (
      <rect
        key={`${cx}-${cy}`}
        x={cx - 1.5}
        y={cy - 1.5}
        width={3}
        height={3}
        className={styles.chromeFill}
      />
    ))}
    {/* Pre-seeded history texture (dim), with a sprinkle of accent entries */}
    {HISTORY.fg.map((d, b) => (
      <path key={b} d={d} className={styles.histCells} fillOpacity={HIST_OPACITIES[b]} />
    ))}
    <path d={HISTORY.blue} className={styles.histAccent} />
    <path d={HISTORY.green} className={styles.histAccentGreen} />
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

const CAPTION =
  "Schedule a run for any point in the future.";

const ARIA_LABEL =
  "Animated diagram of future scheduling in Hatchet: runs are posted onto a timeline at future trigger times, wait as durable schedule entries while a NOW cursor advances, then fire into a worker exactly when their time arrives and settle into run history.";

export const ScheduleTimeline = ({ style }: { style?: CSSProperties }) => (
  <div className={styles.wrapper} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.flowRoot}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={16} y={6} anchor="left">
          schedules · durable
        </StageLabel>
        {MAJORS_LABELLED.map(([x, label]) => (
          <StageLabel key={x} x={x} y={78} muted>
            {label}
          </StageLabel>
        ))}
        <StageLabel x={WORKER_X} y={108}>
          worker
        </StageLabel>
        <StageLabel x={24} y={157} anchor="left" muted>
          completed runs
        </StageLabel>
        <Flow.Token track={WORKER}>
          <div className={styles.workerBox} />
        </Flow.Token>
        <Flow.Token track={CURSOR_LINE}>
          <div className={styles.cursorLine} />
        </Flow.Token>
        <Flow.Token track={CURSOR_TAG}>
          <div className={styles.nowTag}>now</div>
        </Flow.Token>
        {RECORDS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.record} />
          </Flow.Token>
        ))}
        {RESIDUES.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.residue} />
          </Flow.Token>
        ))}
        {PAYLOADS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.square} />
          </Flow.Token>
        ))}
      </Flow.Stage>
    </Flow.Root>
    <Text.Small as="p" secondary balance className={`${styles.caption}`}>
      {CAPTION}
    </Text.Small>
  </div>
);
