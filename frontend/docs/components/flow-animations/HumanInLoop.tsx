"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  defineTrack,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import styles from "./humaninloop.module.css";

/**
 * Human-in-the-loop, as one durable event wait between two labeled regions:
 *
 * The HATCHET ENGINE (left, source of truth) dispatches an agent task to the
 * WORKER (right, stateless muscle — two slots and a load meter). The task
 * runs, hits `wait: approval`, hollows out, leaves the worker entirely, and
 * parks as a durable record in the engine's state list. The freed slot is
 * visibly empty — another task takes it and completes — and while the human
 * deliberates the worker dims to idle: no slot held, nothing burned.
 *
 * The approval arrives from OUTSIDE the system: a human's `approve` chip
 * drops down the inbound line into the engine, resolves the record, and the
 * engine re-dispatches the task to a free slot, where it completes.
 *
 * Everything is scripted data at module scope, so the loop, the SSR poster,
 * and reduced motion all render deterministically.
 */

// ─── Geometry (stage design units, 320 × 216 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 216;

/** Hatchet engine box (left): a durable-state record row, queue strip below. */
const ENGINE = { x: 10, y: 52, w: 136, h: 134 };
const ROW_Y = 86; // durable-state record row (center)
const ROW_RECT = { x: 20, w: 118, h: 18 };
const ROW_SPOT_X = 28; // where the parked task square sits in its row
const ROW_TEXT_X = 36; // record text left edge
const NOTE_X = (ROW_RECT.x + ROW_RECT.x + ROW_RECT.w) / 2; // dwell annotation
const NOTE_Y = 107;
const DIVIDER_Y = 126;
const QUEUE_Y = 158; // queue strip / dispatch connector row
const QUEUE_SPAWN_X = 32;

/** Worker box (right): two slots plus a load meter along the bottom. */
const WORKER = { x: 204, y: 64, w: 108, h: 112 };
const SLOT_X = 258;
const SLOT_YS = [96, 140] as const;
const SLOT_W = 84;
const SLOT_H = 22;
const METER_Y = 164;
const METER_XS = [240, 276] as const; // one segment per slot

/**
 * Inbound human line dropping from off-stage into the ENGINE — right of the
 * engine's section labels so the descending chip never crosses text.
 */
const HUMAN_X = 104;

// ─── Timing (ms) — one scripted approval loop ──────────────────────────────

const DURATION = 12800;

const A_SPAWN = 300;
const A_ARRIVE = 1300; // agent task running in the top slot
const A_WAIT = 2800; // hits wait: approval → hollows out
const A_DEPART = 3050; // leaves the worker
const A_PARKED = 3900; // settles into the engine's durable-state row

const F_SPAWN = 4100; // other work takes the freed slot immediately
const F_ARRIVE = 4900;
const F_DONE = 6400;

const APPROVE_SPAWN = 7600; // the human decides — worker fully idle by now
const APPROVE_STRIKE = 8700; // the chip resolves the record AT the engine
const A_DISPATCH = 9050; // engine re-dispatches
const A_BACK = 9900; // running again (bottom slot)
const A_DONE = 11000;

/** Poster: task parked as a record, other work busy in the freed slot. */
const POSTER_TIME = 5600;

// ─── Track builders ────────────────────────────────────────────────────────

/** Appear on the engine's queue strip, dwell, then get dispatched to a slot. */
const enter = (
  spawn: number,
  arrive: number,
  slotY: number,
): FlowKeyframe[] => [
  { t: spawn, x: QUEUE_SPAWN_X, y: QUEUE_Y, opacity: 0, state: "queued" },
  { t: spawn + 250, x: QUEUE_SPAWN_X, y: QUEUE_Y, opacity: 1, ease: "linear" },
  { t: spawn + 400, x: QUEUE_SPAWN_X, y: QUEUE_Y, ease: "hold" },
  { t: arrive - 250, x: 196, y: QUEUE_Y, ease: "linear" },
  { t: arrive, x: SLOT_X, y: slotY, ease: "out", state: "active" },
];

/** Flash done (green), dwell, then scale-fade out of the slot. */
const complete = (doneAt: number, slotY: number): FlowKeyframe[] => [
  { t: doneAt, x: SLOT_X, y: slotY, ease: "hold", state: "done" },
  { t: doneAt + 350, x: SLOT_X, y: slotY, ease: "hold" },
  { t: doneAt + 650, x: SLOT_X, y: slotY, opacity: 0, scale: 1.5, ease: "out" },
];

/** The agent task: run → park as a durable record → approved → resume → done. */
const TASK_TRACK = defineTrack("hil-task", [
  ...enter(A_SPAWN, A_ARRIVE, SLOT_YS[0]),
  // Hits wait: approval — hollows out; from here on it burns nothing
  { t: A_WAIT, x: SLOT_X, y: SLOT_YS[0], ease: "hold", state: "waiting" },
  // Leaves the worker and slides into the engine's durable-state row
  { t: A_DEPART, x: SLOT_X, y: SLOT_YS[0], ease: "hold" },
  { t: 3450, x: 156, y: 110, ease: "inOut" },
  { t: A_PARKED, x: ROW_SPOT_X, y: ROW_Y, ease: "out", state: "parked" },
  // The approval resolves the record at the engine → re-dispatch
  {
    t: APPROVE_STRIKE,
    x: ROW_SPOT_X,
    y: ROW_Y,
    ease: "hold",
    state: "resuming",
  },
  { t: A_DISPATCH, x: ROW_SPOT_X, y: ROW_Y, ease: "hold" },
  { t: A_DISPATCH + 250, x: 146, y: ROW_Y, ease: "linear" },
  { t: A_BACK, x: SLOT_X, y: SLOT_YS[1], ease: "out", state: "active" },
  ...complete(A_DONE, SLOT_YS[1]),
]);

/** Other work flowing into the slot the waiting task vacated. */
const FILLER_TRACK = defineTrack("hil-filler", [
  ...enter(F_SPAWN, F_ARRIVE, SLOT_YS[0]),
  ...complete(F_DONE, SLOT_YS[0]),
]);

/** Mono record text stamped into the row while the engine tracks the wait. */
const RECORD_TRACK = defineTrack("hil-record", [
  { t: A_PARKED + 100, x: ROW_TEXT_X, y: ROW_Y, opacity: 0 },
  { t: A_PARKED + 350, x: ROW_TEXT_X, y: ROW_Y, opacity: 1, ease: "linear" },
  {
    t: APPROVE_STRIKE,
    x: ROW_TEXT_X,
    y: ROW_Y,
    ease: "hold",
    state: "resolved",
  },
  {
    t: APPROVE_STRIKE + 450,
    x: ROW_TEXT_X,
    y: ROW_Y,
    opacity: 0,
    ease: "linear",
  },
]);

/** Dwell annotation: the whole point — parked work holds no worker slot. */
const NOTE_TRACK = defineTrack("hil-note", [
  { t: A_PARKED + 600, x: NOTE_X, y: NOTE_Y, opacity: 0 },
  { t: A_PARKED + 900, x: NOTE_X, y: NOTE_Y, opacity: 1, ease: "linear" },
  { t: APPROVE_STRIKE - 150, x: NOTE_X, y: NOTE_Y, ease: "hold" },
  { t: APPROVE_STRIKE + 100, x: NOTE_X, y: NOTE_Y, opacity: 0, ease: "linear" },
]);

/** The human's approval: drops down the inbound line into the record. */
const APPROVE_TRACK = defineTrack("hil-approve", [
  { t: APPROVE_SPAWN, x: HUMAN_X, y: 14, opacity: 0, state: "event" },
  { t: APPROVE_SPAWN + 250, x: HUMAN_X, y: 20, opacity: 1, ease: "linear" },
  { t: APPROVE_STRIKE - 200, x: HUMAN_X, y: 60, ease: "in" },
  { t: APPROVE_STRIKE, x: HUMAN_X, y: ROW_Y, ease: "in" },
  { t: APPROVE_STRIKE + 200, x: HUMAN_X, y: ROW_Y, opacity: 0, ease: "linear" },
]);

// ─── Worker state: slot occupancy, load meter, idle dimming ────────────────

type Window = [number, number];

/** Per-slot busy windows — drive both the slot tint and the load meter. */
const SLOT_WINDOWS: Window[][] = [
  [
    [A_ARRIVE, A_DEPART],
    [F_ARRIVE, F_DONE + 400],
  ],
  [[A_BACK, A_DONE + 350]],
];

/** Union of all slot windows: the worker frame dims when fully idle. */
const WORKER_WINDOWS: Window[] = (() => {
  const all = SLOT_WINDOWS.flat().sort((a, b) => a[0] - b[0]);
  const merged: Window[] = [];
  for (const [start, end] of all) {
    const last = merged[merged.length - 1];
    if (last && start <= last[1]) last[1] = Math.max(last[1], end);
    else merged.push([start, end]);
  }
  return merged;
})();

const stateTrack = (
  id: string,
  x: number,
  y: number,
  windows: Window[],
): FlowTrack =>
  defineTrack(id, [
    { t: 0, x, y, state: "idle" },
    ...windows.flatMap(([a, b]): FlowKeyframe[] => [
      { t: a, x, y, state: "busy", ease: "hold" },
      { t: b, x, y, state: "idle", ease: "hold" },
    ]),
    { t: DURATION, x, y, ease: "hold" },
  ]);

const SLOT_TRACKS = SLOT_WINDOWS.map((w, i) =>
  stateTrack(`hil-slot-${i}`, SLOT_X, SLOT_YS[i], w),
);
const METER_TRACKS = SLOT_WINDOWS.map((w, i) =>
  stateTrack(`hil-meter-${i}`, METER_XS[i], METER_Y, w),
);
const WORKER_TRACK = stateTrack(
  "hil-worker-frame",
  WORKER.x + WORKER.w / 2,
  WORKER.y + WORKER.h / 2,
  WORKER_WINDOWS,
);

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
    {/* Hatchet engine box with blueprint corner marks */}
    <rect
      x={ENGINE.x}
      y={ENGINE.y}
      width={ENGINE.w}
      height={ENGINE.h}
      className={styles.chromeStroke}
      {...stroke}
    />
    {[
      [ENGINE.x, ENGINE.y],
      [ENGINE.x + ENGINE.w, ENGINE.y],
      [ENGINE.x, ENGINE.y + ENGINE.h],
      [ENGINE.x + ENGINE.w, ENGINE.y + ENGINE.h],
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
    {/* Durable-state record row (an empty ledger slot until filled) */}
    <rect
      x={ROW_RECT.x}
      y={ROW_Y - ROW_RECT.h / 2}
      width={ROW_RECT.w}
      height={ROW_RECT.h}
      className={styles.chromeRow}
      {...fine}
      strokeDasharray="2 3"
    />
    {/* Divider between state list and queue */}
    <line
      x1={18}
      y1={DIVIDER_Y}
      x2={138}
      y2={DIVIDER_Y}
      className={styles.chromeDash}
      {...fine}
    />
    {/* Queue strip + dispatch connector to the worker */}
    <rect
      x={20}
      y={QUEUE_Y - 2}
      width={4}
      height={4}
      className={styles.chromeFill}
    />
    <line
      x1={28}
      y1={QUEUE_Y}
      x2={140}
      y2={QUEUE_Y}
      className={styles.chromeDash}
      {...fine}
    />
    <line
      x1={152}
      y1={QUEUE_Y}
      x2={196}
      y2={QUEUE_Y}
      className={styles.chromeDash}
      {...fine}
    />
    {/* Worker slot rows (the frame itself is a state-driven token) */}
    {SLOT_YS.map((y) => (
      <rect
        key={y}
        x={SLOT_X - SLOT_W / 2}
        y={y - SLOT_H / 2}
        width={SLOT_W}
        height={SLOT_H}
        className={styles.chromeSlot}
        {...fine}
      />
    ))}
    {/* Inbound human line from off-stage into the engine */}
    <rect
      x={HUMAN_X - 2}
      y={4}
      width={4}
      height={4}
      className={styles.chromeFill}
    />
    <line
      x1={HUMAN_X}
      y1={12}
      x2={HUMAN_X}
      y2={46}
      className={styles.chromeDash}
      {...fine}
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
  "Animated diagram of a human-in-the-loop wait: an agent task runs on a worker, hits wait for approval, leaves the worker, and parks as a durable record on the Hatchet engine. The freed slot takes other work and the worker then dims to idle, consuming nothing, while the task waits. A human approval arrives at the engine, resolves the record, and the engine re-dispatches the task to a free slot, where it completes.";

interface HumanInLoopProps {
  className?: string;
  style?: CSSProperties;
}

export const HumanInLoop = ({ className, style }: HumanInLoopProps) => (
  <div className={`${styles.wrap} ${className ?? ""}`} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.root}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={12} y={42} anchor="left">
          hatchet engine
        </StageLabel>
        <StageLabel x={18} y={62} anchor="left" muted>
          durable state
        </StageLabel>
        <StageLabel x={18} y={144} anchor="left" muted>
          queue
        </StageLabel>
        <StageLabel x={HUMAN_X + 8} y={5} anchor="left" muted>
          human
        </StageLabel>
        <StageLabel x={SLOT_X} y={196}>
          worker
        </StageLabel>
        <StageLabel x={212} y={161} anchor="left" muted>
          load
        </StageLabel>
        <Flow.Token track={WORKER_TRACK}>
          <div className={styles.workerFrame} />
        </Flow.Token>
        {SLOT_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.slotFill} />
          </Flow.Token>
        ))}
        {METER_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.meterSeg} />
          </Flow.Token>
        ))}
        <Flow.Token track={APPROVE_TRACK}>
          <div className={styles.approveChip}>approve</div>
        </Flow.Token>
        <Flow.Token track={RECORD_TRACK}>
          <div className={styles.record}>wait: approval</div>
        </Flow.Token>
        <Flow.Token track={NOTE_TRACK}>
          <div className={styles.note}>no slot held</div>
        </Flow.Token>
        <Flow.Token track={FILLER_TRACK}>
          <div className={styles.square} />
        </Flow.Token>
        <Flow.Token track={TASK_TRACK}>
          <div className={styles.square} />
        </Flow.Token>
      </Flow.Stage>
    </Flow.Root>
  </div>
);
