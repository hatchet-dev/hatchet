"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  createSeededRandom,
  defineTrack,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import styles from "./durablewaits.module.css";

/**
 * Durable waits, in two beats between two labeled regions:
 *
 * The HATCHET ENGINE (left, source of truth) queues tasks and dispatches them
 * to the WORKER (right, stateless muscle — three slots and a load meter).
 *
 * Beat 1 — sleep. A task runs in a slot, hits its durable sleep, hollows out,
 * leaves the worker entirely, and slides into the engine's durable-state list
 * as a `sleep(24h)` record with a depleting tick countdown. The freed slot is
 * visibly empty, the load meter drops, and the worker picks up OTHER tasks
 * while the sleeper costs nothing. When the countdown elapses (at the engine),
 * the engine re-dispatches the task to a free slot and it completes.
 *
 * Beat 2 — event wait. A second task parks the same way as an `on: approval`
 * record. The worker finishes its other work and dims to idle — burning
 * nothing — while the approval event arrives AT THE ENGINE, resolves the
 * record, and the engine re-dispatches the task to complete.
 *
 * Everything is scripted data at module scope (seeded jitter only), so the
 * loop, the SSR poster, and reduced motion all render deterministically.
 */

// ─── Geometry (stage design units, 320 × 224 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 224;

/** Hatchet engine box (left): durable-state rows on top, queue strip below. */
const ENGINE = { x: 10, y: 46, w: 130, h: 142 };
const ROW_YS = [74, 96] as const; // durable-state record rows (centers)
const ROW_RECT = { x: 20, w: 112, h: 18 };
const ROW_SPOT_X = 26; // where a tracked task square sits in its row
const ROW_TEXT_X = 34; // record text left edge
const QUEUE_Y = 160; // queue strip / dispatch connector row
const QUEUE_SPAWN_X = 34;
const DIVIDER_Y = 122;

/** Worker box (right): three slots plus a load meter along the bottom. */
const WORKER = { x: 200, y: 64, w: 112, h: 136 };
const SLOT_X = 256;
const SLOT_YS = [88, 124, 160] as const;
const SLOT_W = 80;
const SLOT_H = 20;
const METER_Y = 188;
const METER_XS = [242, 264, 286] as const; // one segment per slot

/**
 * Inbound event line dropping from off-stage into the ENGINE — right of the
 * engine's section labels so the descending token never crosses text.
 */
const EVENT_X = 108;

/** Countdown ticks beside the sleep record. */
const TICK_X0 = 82;
const TICK_PITCH = 7;

// ─── Timing (ms) — a scripted two-beat loop ────────────────────────────────

const DURATION = 16000;

// Beat 1 — durable sleep (task A)
const A_SPAWN = 300;
const A_ARRIVE = 1300; // running in the middle slot
const A_WAIT = 2600; // hits sleep(24h) → hollows out
const A_DEPART = 2850; // leaves the worker
const A_TRACKED = 3700; // settles into the engine's state list (row 2)
const TICK_FADES = [4700, 5500, 6300, 7100];
const A_RESUME = 7100; // countdown elapsed at the engine
const A_DISPATCH = 7400; // engine re-dispatches
const A_BACK = 8100; // running again (top slot)
const A_DONE = 9100;

// Beat 2 — durable event wait (task B)
const B_SPAWN = 8800;
const B_ARRIVE = 9500; // bottom slot
const B_WAIT = 10800; // establishes on: approval
const B_DEPART = 11050;
const B_TRACKED = 12000; // row 1
const EVENT_SPAWN = 12600;
const EVENT_STRIKE = 13600; // the event resolves the record AT the engine
const B_DISPATCH = 13900;
const B_BACK = 14600; // resumes in the freed middle slot
const B_DONE = 15100;

/** Poster: A tracked in the engine + a filler busy in A's freed slot (load 1/3). */
const POSTER_TIME = 4800;

// ─── Filler tasks (the worker stays free for other work) ───────────────────

const rng = createSeededRandom("durable-waits");

interface Filler {
  spawn: number;
  slot: number;
  arrive: number;
  done: number;
  fadeStart: number;
}

const FILLERS: Filler[] = [
  { spawn: 3600, slot: 1 }, // straight into the slot A vacated
  { spawn: 6000, slot: 1 },
  { spawn: 10700, slot: 2 }, // …and into the slot B vacated
].map((f) => {
  const arrive = f.spawn + 800;
  const done = arrive + Math.round(1150 + rng() * 350);
  return { ...f, arrive, done, fadeStart: done + 400 };
});

// ─── Track builders ────────────────────────────────────────────────────────

/** Appear on the engine's queue strip, dwell, then get dispatched to a slot. */
const enter = (spawn: number, arrive: number, slotY: number): FlowKeyframe[] => [
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

const A_TRACK = defineTrack("task-sleep", [
  ...enter(A_SPAWN, A_ARRIVE, SLOT_YS[1]),
  // Hits sleep(24h): hollows out — from here on it burns nothing
  { t: A_WAIT, x: SLOT_X, y: SLOT_YS[1], ease: "hold", state: "waiting" },
  // Leaves the worker and slides into the engine's durable-state row
  { t: A_DEPART, x: SLOT_X, y: SLOT_YS[1], ease: "hold" },
  { t: 3200, x: 150, y: ROW_YS[1], ease: "inOut" },
  { t: A_TRACKED, x: ROW_SPOT_X, y: ROW_YS[1], ease: "out", state: "tracked" },
  // Countdown elapses at the engine → re-dispatch to a free slot
  { t: A_RESUME, x: ROW_SPOT_X, y: ROW_YS[1], ease: "hold", state: "resuming" },
  { t: A_DISPATCH, x: ROW_SPOT_X, y: ROW_YS[1], ease: "hold" },
  { t: A_DISPATCH + 250, x: 140, y: ROW_YS[1], ease: "linear" },
  { t: A_BACK, x: SLOT_X, y: SLOT_YS[0], ease: "out", state: "active" },
  ...complete(A_DONE, SLOT_YS[0]),
]);

const B_TRACK = defineTrack("task-event", [
  ...enter(B_SPAWN, B_ARRIVE, SLOT_YS[2]),
  { t: B_WAIT, x: SLOT_X, y: SLOT_YS[2], ease: "hold", state: "waiting" },
  { t: B_DEPART, x: SLOT_X, y: SLOT_YS[2], ease: "hold" },
  { t: 11350, x: 170, y: 120, ease: "inOut" },
  { t: 11650, x: 150, y: ROW_YS[0], ease: "inOut" },
  { t: B_TRACKED, x: ROW_SPOT_X, y: ROW_YS[0], ease: "out", state: "tracked" },
  // The approval event resolves the record at the engine → re-dispatch
  { t: EVENT_STRIKE, x: ROW_SPOT_X, y: ROW_YS[0], ease: "hold", state: "resuming" },
  { t: B_DISPATCH, x: ROW_SPOT_X, y: ROW_YS[0], ease: "hold" },
  { t: B_DISPATCH + 250, x: 140, y: ROW_YS[0], ease: "linear" },
  { t: B_BACK, x: SLOT_X, y: SLOT_YS[1], ease: "out", state: "active" },
  ...complete(B_DONE, SLOT_YS[1]),
]);

const FILLER_TRACKS: FlowTrack[] = FILLERS.map((f, i) =>
  defineTrack(`filler-${i}`, [
    ...enter(f.spawn, f.arrive, SLOT_YS[f.slot]),
    ...complete(f.done, SLOT_YS[f.slot]),
  ])
);

/** Depleting countdown beside the sleep record (compressed time). */
const TICK_TRACKS: FlowTrack[] = TICK_FADES.map((fade, i) => {
  const x = TICK_X0 + i * TICK_PITCH;
  return defineTrack(`tick-${i}`, [
    { t: A_TRACKED + 250, x, y: ROW_YS[1], opacity: 0 },
    { t: A_TRACKED + 500 + i * 120, x, y: ROW_YS[1], opacity: 1, ease: "linear" },
    { t: fade - 250, x, y: ROW_YS[1], ease: "hold" },
    { t: fade, x, y: ROW_YS[1], opacity: 0, ease: "linear" },
  ]);
});

/** The approval event: drops down the inbound line into the engine's record. */
const EVENT_TRACK = defineTrack("event", [
  { t: EVENT_SPAWN, x: EVENT_X, y: 10, opacity: 0, state: "event" },
  { t: EVENT_SPAWN + 250, x: EVENT_X, y: 16, opacity: 1, ease: "linear" },
  { t: EVENT_STRIKE - 200, x: EVENT_X, y: 56, ease: "in" },
  { t: EVENT_STRIKE, x: EVENT_X, y: ROW_YS[0], ease: "in" },
  { t: EVENT_STRIKE + 200, x: EVENT_X, y: ROW_YS[0], opacity: 0, ease: "linear" },
]);

/** Mono record text stamped into a row while the engine tracks the wait. */
const recordTrack = (
  id: string,
  rowY: number,
  showAt: number,
  resolveAt: number
): FlowTrack =>
  defineTrack(id, [
    { t: showAt, x: ROW_TEXT_X, y: rowY, opacity: 0 },
    { t: showAt + 250, x: ROW_TEXT_X, y: rowY, opacity: 1, ease: "linear" },
    { t: resolveAt, x: ROW_TEXT_X, y: rowY, ease: "hold", state: "resolved" },
    { t: resolveAt + 350, x: ROW_TEXT_X, y: rowY, opacity: 0, ease: "linear" },
  ]);

const SLEEP_RECORD = recordTrack("record-sleep", ROW_YS[1], A_TRACKED + 100, A_RESUME);
const APPROVAL_RECORD = recordTrack("record-approval", ROW_YS[0], B_TRACKED + 100, EVENT_STRIKE);

// ─── Worker state: slot occupancy, load meter, idle dimming ────────────────

type Window = [number, number];

const fillerWindows = (slot: number): Window[] =>
  FILLERS.filter((f) => f.slot === slot).map((f) => [f.arrive, f.fadeStart]);

/** Per-slot busy windows — drive both the slot tint and the load meter. */
const SLOT_WINDOWS: Window[][] = [
  [[A_BACK, A_DONE + 350]],
  [[A_ARRIVE, A_DEPART], ...fillerWindows(1), [B_BACK, B_DONE + 350]],
  [[B_ARRIVE, B_DEPART], ...fillerWindows(2)],
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

const stateTrack = (id: string, x: number, y: number, windows: Window[]): FlowTrack =>
  defineTrack(id, [
    { t: 0, x, y, state: "idle" },
    ...windows.flatMap(([a, b]): FlowKeyframe[] => [
      { t: a, x, y, state: "busy", ease: "hold" },
      { t: b, x, y, state: "idle", ease: "hold" },
    ]),
    { t: DURATION, x, y, ease: "hold" },
  ]);

const SLOT_TRACKS = SLOT_WINDOWS.map((w, i) => stateTrack(`slot-${i}`, SLOT_X, SLOT_YS[i], w));
const METER_TRACKS = SLOT_WINDOWS.map((w, i) => stateTrack(`meter-${i}`, METER_XS[i], METER_Y, w));
const WORKER_TRACK = stateTrack(
  "worker-frame",
  WORKER.x + WORKER.w / 2,
  WORKER.y + WORKER.h / 2,
  WORKER_WINDOWS
);

// ─── Static chrome ─────────────────────────────────────────────────────────

const stroke = { fill: "none", strokeWidth: 1.5, vectorEffect: "non-scaling-stroke" } as const;
const fine = { fill: "none", strokeWidth: 1, vectorEffect: "non-scaling-stroke" } as const;

const Chrome = () => (
  <svg viewBox={`0 0 ${STAGE_W} ${STAGE_H}`} aria-hidden="true" className={styles.chrome}>
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
    {/* Durable-state record rows (empty ledger slots until filled) */}
    {ROW_YS.map((y) => (
      <rect
        key={y}
        x={ROW_RECT.x}
        y={y - ROW_RECT.h / 2}
        width={ROW_RECT.w}
        height={ROW_RECT.h}
        className={styles.chromeRow}
        {...fine}
        strokeDasharray="2 3"
      />
    ))}
    {/* Divider between state list and queue */}
    <line x1={18} y1={DIVIDER_Y} x2={132} y2={DIVIDER_Y} className={styles.chromeDash} {...fine} />
    {/* Queue strip + dispatch connector to the worker */}
    <rect x={20} y={QUEUE_Y - 2} width={4} height={4} className={styles.chromeFill} />
    <line x1={28} y1={QUEUE_Y} x2={134} y2={QUEUE_Y} className={styles.chromeDash} {...fine} />
    <line x1={146} y1={QUEUE_Y} x2={196} y2={QUEUE_Y} className={styles.chromeDash} {...fine} />
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
    {/* Inbound event line from off-stage into the engine */}
    <rect x={EVENT_X - 2} y={4} width={4} height={4} className={styles.chromeFill} />
    <line x1={EVENT_X} y1={12} x2={EVENT_X} y2={40} className={styles.chromeDash} {...fine} />
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
  "Animated diagram of durable waits: a task hits sleep(24h), leaves the worker, and is tracked as a durable record on the Hatchet engine while the freed slot takes other tasks and the worker's load drops. When the countdown elapses the engine re-dispatches it and it completes. A second task waits for an approval event the same way; the worker sits idle burning nothing until the event arrives at the engine, which re-dispatches the task to completion.";

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
        <StageLabel x={12} y={36} anchor="left">
          hatchet engine
        </StageLabel>
        <StageLabel x={18} y={56} anchor="left" muted>
          durable state
        </StageLabel>
        <StageLabel x={18} y={146} anchor="left" muted>
          queue
        </StageLabel>
        <StageLabel x={EVENT_X + 8} y={5} anchor="left" muted>
          event
        </StageLabel>
        <StageLabel x={SLOT_X} y={208}>
          worker
        </StageLabel>
        <StageLabel x={214} y={185} anchor="left" muted>
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
        <Flow.Token track={EVENT_TRACK}>
          <div className={styles.eventDot} />
        </Flow.Token>
        <Flow.Token track={SLEEP_RECORD}>
          <div className={styles.record}>sleep(24h)</div>
        </Flow.Token>
        <Flow.Token track={APPROVAL_RECORD}>
          <div className={styles.record}>on: approval</div>
        </Flow.Token>
        {FILLER_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.square} />
          </Flow.Token>
        ))}
        <Flow.Token track={A_TRACK}>
          <div className={styles.square} />
        </Flow.Token>
        <Flow.Token track={B_TRACK}>
          <div className={styles.square} />
        </Flow.Token>
      </Flow.Stage>
    </Flow.Root>
  </div>
);
