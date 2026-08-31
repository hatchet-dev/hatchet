"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  defineTrack,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import styles from "./durablewaits.module.css";

/**
 * Durable sleep, as one legible beat sequence between two labeled regions:
 *
 * The HATCHET ENGINE (left, source of truth) dispatches a task to the WORKER
 * (right, stateless muscle — two slots and a load meter). The task runs,
 * calls `sleep(24h)` — the call itself flashes as a mono chip at the task —
 * hollows out, leaves the worker entirely, and parks as a durable
 * `sleep(24h)` record in the engine's state list. The freed slot is visibly
 * empty — another task takes it and completes — while a tick countdown
 * beside the record depletes: time passing, no worker slot held.
 *
 * When the countdown elapses, the record resolves AT the engine, which
 * re-dispatches the task to a free slot, where it completes.
 *
 * Everything is scripted data at module scope, so the loop, the SSR poster,
 * and reduced motion all render deterministically.
 */

// ─── Geometry (stage design units, 320 × 208 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 208;

/** Hatchet engine box (left): a durable-state record row, queue strip below. */
const ENGINE = { x: 10, y: 44, w: 136, h: 134 };
const ROW_Y = 78; // durable-state record row (center)
const ROW_RECT = { x: 20, w: 118, h: 18 };
const ROW_SPOT_X = 28; // where the parked task square sits in its row
const ROW_TEXT_X = 36; // record text left edge
const NOTE_X = (ROW_RECT.x + ROW_RECT.x + ROW_RECT.w) / 2; // dwell annotation
const NOTE_Y = 99;
const DIVIDER_Y = 118;
const QUEUE_Y = 150; // queue strip / dispatch connector row
const QUEUE_SPAWN_X = 32;

/** Countdown ticks inside the record row, right of the record text. */
const TICK_X0 = 88;
const TICK_PITCH = 7;

/** Worker box (right): two slots plus a load meter along the bottom. */
const WORKER = { x: 204, y: 56, w: 108, h: 112 };
const SLOT_X = 258;
const SLOT_YS = [88, 132] as const;
const SLOT_W = 84;
const SLOT_H = 22;
const METER_Y = 156;
const METER_XS = [240, 276] as const; // one segment per slot

/** The `sleep(24h)` call chip, flashed just above the running task. */
const CHIP_X = SLOT_X;
const CHIP_Y = SLOT_YS[0] - 17;

// ─── Timing (ms) — one scripted sleep loop ─────────────────────────────────

const DURATION = 13200;

const A_SPAWN = 300;
const A_ARRIVE = 1300; // task running in the top slot
const CHIP_IN = 2600; // the task calls sleep(24h)
const A_WAIT = 3000; // the call takes effect → the task hollows out
const A_DEPART = 3450; // leaves the worker
const CHIP_OUT = 3900;
const A_PARKED = 4300; // settles into the engine's durable-state row

const F_SPAWN = 4600; // other work takes the freed slot
const F_ARRIVE = 5400;
const F_DONE = 7000;

/** Countdown ticks deplete one by one — 24 hours in compressed time. */
const TICK_FADES = [6000, 7000, 8000, 9000];

const A_RESUME = 9000; // the timer elapses at the engine
const A_DISPATCH = 9400; // engine re-dispatches
const A_BACK = 10300; // running again (bottom slot)
const A_DONE = 11500;

/** Poster: task parked as a sleep record (countdown partly depleted), other
 * work busy in the freed slot. */
const POSTER_TIME = 6400;

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

/** The task: run → sleep(24h) → park as a durable record → resume → done. */
const TASK_TRACK = defineTrack("dw-task", [
  ...enter(A_SPAWN, A_ARRIVE, SLOT_YS[0]),
  // Calls sleep(24h) — hollows out; from here on it burns nothing
  { t: A_WAIT, x: SLOT_X, y: SLOT_YS[0], ease: "hold", state: "waiting" },
  // Leaves the worker and slides into the engine's durable-state row
  { t: A_DEPART, x: SLOT_X, y: SLOT_YS[0], ease: "hold" },
  { t: 3850, x: 156, y: 102, ease: "inOut" },
  { t: A_PARKED, x: ROW_SPOT_X, y: ROW_Y, ease: "out", state: "parked" },
  // The countdown elapses at the engine → re-dispatch
  { t: A_RESUME, x: ROW_SPOT_X, y: ROW_Y, ease: "hold", state: "resuming" },
  { t: A_DISPATCH, x: ROW_SPOT_X, y: ROW_Y, ease: "hold" },
  { t: A_DISPATCH + 250, x: 146, y: ROW_Y, ease: "linear" },
  { t: A_BACK, x: SLOT_X, y: SLOT_YS[1], ease: "out", state: "active" },
  ...complete(A_DONE, SLOT_YS[1]),
]);

/** Other work flowing into the slot the sleeping task vacated. */
const FILLER_TRACK = defineTrack("dw-filler", [
  ...enter(F_SPAWN, F_ARRIVE, SLOT_YS[0]),
  ...complete(F_DONE, SLOT_YS[0]),
]);

/** The call itself: a mono `sleep(24h)` chip flashed at the running task. */
const CHIP_TRACK = defineTrack("dw-chip", [
  { t: CHIP_IN, x: CHIP_X, y: CHIP_Y, opacity: 0, scale: 0.85 },
  { t: CHIP_IN + 300, x: CHIP_X, y: CHIP_Y, opacity: 1, scale: 1, ease: "out" },
  { t: CHIP_OUT - 250, x: CHIP_X, y: CHIP_Y, ease: "hold" },
  { t: CHIP_OUT, x: CHIP_X, y: CHIP_Y, opacity: 0, ease: "linear" },
]);

/** Mono record text stamped into the row while the engine tracks the sleep. */
const RECORD_TRACK = defineTrack("dw-record", [
  { t: A_PARKED + 100, x: ROW_TEXT_X, y: ROW_Y, opacity: 0 },
  { t: A_PARKED + 350, x: ROW_TEXT_X, y: ROW_Y, opacity: 1, ease: "linear" },
  { t: A_RESUME, x: ROW_TEXT_X, y: ROW_Y, ease: "hold", state: "resolved" },
  { t: A_RESUME + 450, x: ROW_TEXT_X, y: ROW_Y, opacity: 0, ease: "linear" },
]);

/** Depleting countdown inside the record row (24h in compressed time). */
const TICK_TRACKS: FlowTrack[] = TICK_FADES.map((fade, i) => {
  const x = TICK_X0 + i * TICK_PITCH;
  return defineTrack(`dw-tick-${i}`, [
    { t: A_PARKED + 250, x, y: ROW_Y, opacity: 0 },
    { t: A_PARKED + 500 + i * 120, x, y: ROW_Y, opacity: 1, ease: "linear" },
    { t: fade - 250, x, y: ROW_Y, ease: "hold" },
    { t: fade, x, y: ROW_Y, opacity: 0, ease: "linear" },
  ]);
});

/** Dwell annotation: the whole point — sleeping work holds no worker slot. */
const NOTE_TRACK = defineTrack("dw-note", [
  { t: A_PARKED + 600, x: NOTE_X, y: NOTE_Y, opacity: 0 },
  { t: A_PARKED + 900, x: NOTE_X, y: NOTE_Y, opacity: 1, ease: "linear" },
  { t: A_RESUME - 200, x: NOTE_X, y: NOTE_Y, ease: "hold" },
  { t: A_RESUME + 100, x: NOTE_X, y: NOTE_Y, opacity: 0, ease: "linear" },
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
  stateTrack(`dw-slot-${i}`, SLOT_X, SLOT_YS[i], w),
);
const METER_TRACKS = SLOT_WINDOWS.map((w, i) =>
  stateTrack(`dw-meter-${i}`, METER_XS[i], METER_Y, w),
);
const WORKER_TRACK = stateTrack(
  "dw-worker-frame",
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
  "Animated diagram of durable sleep: a task runs on a worker, calls sleep(24h), leaves the worker, and parks as a durable sleep(24h) record on the Hatchet engine. The freed slot takes other work while a countdown beside the record depletes — no worker slot is held during the sleep. When the timer elapses, the engine re-dispatches the task to a free slot, where it completes.";

interface DurableWaitsProps {
  className?: string;
  style?: CSSProperties;
}

export const DurableWaits = ({ className, style }: DurableWaitsProps) => (
  <div className={`${styles.wrap} ${className ?? ""}`} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.root}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={12} y={34} anchor="left">
          hatchet engine
        </StageLabel>
        <StageLabel x={18} y={54} anchor="left" muted>
          durable state
        </StageLabel>
        <StageLabel x={18} y={136} anchor="left" muted>
          queue
        </StageLabel>
        <StageLabel x={SLOT_X} y={176}>
          worker
        </StageLabel>
        <StageLabel x={212} y={153} anchor="left" muted>
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
        {TICK_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.tick} />
          </Flow.Token>
        ))}
        <Flow.Token track={CHIP_TRACK}>
          <div className={styles.callChip}>sleep(24h)</div>
        </Flow.Token>
        <Flow.Token track={RECORD_TRACK}>
          <div className={styles.record}>sleep(24h)</div>
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
