"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  defineTrack,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import styles from "./durableexecution.module.css";

/**
 * Durable execution, told as one crash-and-replay story:
 *
 * A durable task is dispatched from the HATCHET ENGINE (left — its durable
 * event log is the source of truth) to WORKER-A (top right), where it runs
 * step by step across a four-cell lane. Each completed step emits a
 * checkpoint chip that flies into the engine's log and stamps a record.
 *
 * Mid-step-3 the worker dies: the frame flickers out (accent-1), the
 * in-flight step is lost — but the log persists. The engine re-dispatches
 * the task to WORKER-B, which fast-forwards through the checkpointed steps
 * as hollow "replay" beats (quick, dimmed, log rows highlighted as they're
 * read — nothing re-executes), then goes live at step 3, finishes steps
 * 3 and 4 (checkpointing each), and completes.
 *
 * Core beat: completed work is never repeated. Everything is scripted data
 * at module scope, so the loop, the SSR poster, and reduced motion all
 * render deterministically.
 */

// ─── Geometry (stage design units, 320 × 224 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 224;

/** Hatchet engine box (left): durable event log rows on top, queue below. */
const ENGINE = { x: 10, y: 34, w: 124, h: 146 };
const ROW_YS = [66, 86, 106, 126] as const; // log record rows (centers)
const ROW_RECT = { x: 18, w: 108, h: 16 };
const CHIP_X = 26; // where a checkpoint chip parks in its row
const TEXT_X = 36; // record text left edge
const DIVIDER_Y = 142;
const QUEUE_Y = 162; // queue strip / dispatch connector row
const QUEUE_SPAWN_X = 30;

/** Two worker boxes (right), each with a four-cell step lane. */
const WORKER_W = 114;
const WORKER_H = 68;
const WORKER_X = 196;
const WORKER_A = { x: WORKER_X, y: 34 };
const WORKER_B = { x: WORKER_X, y: 112 };
const CELL_XS = [214, 240, 266, 292] as const; // step cell centers
const CELL_W = 22;
const A_CELL_Y = 78;
const B_CELL_Y = 156;

/** Beat annotations, subtitle-style, on the strip below both regions. */
const BEAT_X = 160;
const BEAT_Y = 200;

// ─── Timing (ms) — one scripted crash-and-replay loop ──────────────────────

const DURATION = 15000;

// First run on worker-a
const SPAWN = 300;
const ARRIVE_A = 1400; // running step 1
const STEP1_DONE = 2600; // → checkpoint chip 1 flies to the log
const STEP2_DONE = 4200; // → checkpoint chip 2
const CRASH = 5700; // worker-a dies mid-step-3; the log persists

// Re-dispatch to worker-b, replay from the log, then live
const RESPAWN = 7000;
const ARRIVE_B = 7900; // replaying step 1 (fast, hollow)
const REPLAY1_END = 8400;
const REPLAY2_END = 9050;
const LIVE = 9250; // live again at step 3 — exactly where it left off
const STEP3_DONE = 10500; // → checkpoint chip 3
const STEP4_DONE = 12000; // → checkpoint chip 4, task completes
const CHIP_FLIGHT = 800;

// Wind-down: hold the full log as a beat, then fade for a clean wrap
const LOG_FADE = 13400;
const RESET = 13800;

/** Poster: worker-a dark, log rows 1–2 stamped, task replaying on worker-b. */
const POSTER_TIME = 8700;

// ─── Task tracks ───────────────────────────────────────────────────────────

/** Fade in on the engine's queue strip, dwell, then travel out. */
const dispatch = (spawn: number, depart: number): FlowKeyframe[] => [
  { t: spawn, x: QUEUE_SPAWN_X, y: QUEUE_Y, opacity: 0, state: "queued" },
  { t: spawn + 250, x: QUEUE_SPAWN_X, y: QUEUE_Y, opacity: 1, ease: "linear" },
  { t: spawn + 450, x: QUEUE_SPAWN_X, y: QUEUE_Y, ease: "hold" },
  { t: depart, x: 150, y: QUEUE_Y, ease: "linear" },
];

/** First run: steps 1 and 2 complete, then the worker dies under it. */
const RUN_1 = defineTrack("de-run-1", [
  ...dispatch(SPAWN, 1050),
  { t: ARRIVE_A, x: CELL_XS[0], y: A_CELL_Y, ease: "out", state: "active" },
  { t: STEP1_DONE, x: CELL_XS[0], y: A_CELL_Y, ease: "hold" },
  { t: STEP1_DONE + 250, x: CELL_XS[1], y: A_CELL_Y, ease: "out" },
  { t: STEP2_DONE, x: CELL_XS[1], y: A_CELL_Y, ease: "hold" },
  { t: STEP2_DONE + 250, x: CELL_XS[2], y: A_CELL_Y, ease: "out" },
  // Worker crash: the in-flight step is lost with it
  { t: CRASH, x: CELL_XS[2], y: A_CELL_Y, ease: "hold", state: "lost" },
  { t: CRASH + 600, x: CELL_XS[2], y: A_CELL_Y, opacity: 0, ease: "linear" },
]);

/**
 * Re-dispatched run: fast-forwards through the checkpointed steps as hollow
 * replay beats, then continues live from step 3 and completes.
 */
const RUN_2 = defineTrack("de-run-2", [
  ...dispatch(RESPAWN, 7650),
  { t: ARRIVE_B, x: CELL_XS[0], y: B_CELL_Y, ease: "out", state: "replay" },
  { t: REPLAY1_END, x: CELL_XS[0], y: B_CELL_Y, ease: "hold" },
  { t: REPLAY1_END + 150, x: CELL_XS[1], y: B_CELL_Y, ease: "out" },
  { t: REPLAY2_END, x: CELL_XS[1], y: B_CELL_Y, ease: "hold" },
  // Caught up: live execution resumes exactly where it left off
  { t: LIVE, x: CELL_XS[2], y: B_CELL_Y, ease: "out", state: "active" },
  { t: STEP3_DONE, x: CELL_XS[2], y: B_CELL_Y, ease: "hold" },
  { t: STEP3_DONE + 250, x: CELL_XS[3], y: B_CELL_Y, ease: "out" },
  { t: STEP4_DONE, x: CELL_XS[3], y: B_CELL_Y, ease: "hold", state: "done" },
  { t: STEP4_DONE + 400, x: CELL_XS[3], y: B_CELL_Y, ease: "hold" },
  {
    t: STEP4_DONE + 750,
    x: CELL_XS[3],
    y: B_CELL_Y,
    opacity: 0,
    scale: 1.5,
    ease: "out",
  },
]);

// ─── Checkpoint chips + log records ────────────────────────────────────────

interface Checkpoint {
  doneAt: number; // step completion → chip spawns at the cell
  fromX: number;
  fromY: number;
  rowY: number;
  read?: [number, number]; // replay window: the log row is read, not re-run
}

const CHECKPOINTS: Checkpoint[] = [
  {
    doneAt: STEP1_DONE,
    fromX: CELL_XS[0],
    fromY: A_CELL_Y,
    rowY: ROW_YS[0],
    read: [ARRIVE_B, REPLAY1_END],
  },
  {
    doneAt: STEP2_DONE,
    fromX: CELL_XS[1],
    fromY: A_CELL_Y,
    rowY: ROW_YS[1],
    read: [REPLAY1_END + 150, REPLAY2_END],
  },
  { doneAt: STEP3_DONE, fromX: CELL_XS[2], fromY: B_CELL_Y, rowY: ROW_YS[2] },
  { doneAt: STEP4_DONE, fromX: CELL_XS[3], fromY: B_CELL_Y, rowY: ROW_YS[3] },
];

/** Chip: spawn at the completed cell, fly to the log, persist, fade at reset. */
const CHIP_TRACKS: FlowTrack[] = CHECKPOINTS.map((c, i) => {
  const land = c.doneAt + CHIP_FLIGHT;
  const fade = LOG_FADE + i * 50;
  return defineTrack(`de-chip-${i}`, [
    { t: c.doneAt, x: c.fromX, y: c.fromY, opacity: 0 },
    { t: c.doneAt + 150, x: c.fromX, y: c.fromY, opacity: 1, ease: "linear" },
    { t: land, x: CHIP_X, y: c.rowY, ease: "inOut", state: "write" },
    ...(c.read
      ? ([
          { t: c.read[0], x: CHIP_X, y: c.rowY, ease: "hold", state: "read" },
          {
            t: c.read[1],
            x: CHIP_X,
            y: c.rowY,
            ease: "hold",
            state: "settled",
          },
        ] satisfies FlowKeyframe[])
      : []),
    { t: fade, x: CHIP_X, y: c.rowY, ease: "hold" },
    { t: fade + 500, x: CHIP_X, y: c.rowY, opacity: 0, ease: "linear" },
  ]);
});

/** Mono record stamped into the row as its chip lands. */
const RECORD_TRACKS: FlowTrack[] = CHECKPOINTS.map((c, i) => {
  const land = c.doneAt + CHIP_FLIGHT;
  const fade = LOG_FADE + i * 50;
  return defineTrack(`de-record-${i}`, [
    { t: land, x: TEXT_X, y: c.rowY, opacity: 0 },
    { t: land + 250, x: TEXT_X, y: c.rowY, opacity: 1, ease: "linear" },
    ...(c.read
      ? ([
          { t: c.read[0], x: TEXT_X, y: c.rowY, ease: "hold", state: "read" },
          {
            t: c.read[1],
            x: TEXT_X,
            y: c.rowY,
            ease: "hold",
            state: "settled",
          },
        ] satisfies FlowKeyframe[])
      : []),
    { t: fade, x: TEXT_X, y: c.rowY, ease: "hold" },
    { t: fade + 500, x: TEXT_X, y: c.rowY, opacity: 0, ease: "linear" },
  ]);
});

// ─── State machines: step cells + worker frames ────────────────────────────

const stateTrack = (
  id: string,
  x: number,
  y: number,
  flips: [number, string][],
): FlowTrack =>
  defineTrack(id, [
    { t: 0, x, y, state: "idle" },
    ...flips.map(([t, state]): FlowKeyframe => ({
      t,
      x,
      y,
      state,
      ease: "hold",
    })),
    { t: DURATION, x, y, ease: "hold" },
  ]);

/** Worker-a cells: 1–2 complete, 3 in flight — all lost with the worker. */
const A_CELL_TRACKS = [
  stateTrack("de-cell-a0", CELL_XS[0], A_CELL_Y, [
    [ARRIVE_A, "running"],
    [STEP1_DONE, "done"],
    [CRASH, "dead"],
    [RESET, "idle"],
  ]),
  stateTrack("de-cell-a1", CELL_XS[1], A_CELL_Y, [
    [STEP1_DONE + 250, "running"],
    [STEP2_DONE, "done"],
    [CRASH, "dead"],
    [RESET, "idle"],
  ]),
  stateTrack("de-cell-a2", CELL_XS[2], A_CELL_Y, [
    [STEP2_DONE + 250, "running"],
    [CRASH, "dead"],
    [RESET, "idle"],
  ]),
  stateTrack("de-cell-a3", CELL_XS[3], A_CELL_Y, [
    [CRASH, "dead"],
    [RESET, "idle"],
  ]),
];

/** Worker-b cells: 1–2 restored from the log (dim), 3–4 completed live. */
const B_CELL_TRACKS = [
  stateTrack("de-cell-b0", CELL_XS[0], B_CELL_Y, [
    [REPLAY1_END, "replayed"],
    [LOG_FADE, "idle"],
  ]),
  stateTrack("de-cell-b1", CELL_XS[1], B_CELL_Y, [
    [REPLAY2_END, "replayed"],
    [LOG_FADE, "idle"],
  ]),
  stateTrack("de-cell-b2", CELL_XS[2], B_CELL_Y, [
    [LIVE, "running"],
    [STEP3_DONE, "done"],
    [LOG_FADE + 100, "idle"],
  ]),
  stateTrack("de-cell-b3", CELL_XS[3], B_CELL_Y, [
    [STEP3_DONE + 250, "running"],
    [STEP4_DONE, "done"],
    [LOG_FADE + 100, "idle"],
  ]),
];

const WORKER_A_TRACK = stateTrack(
  "de-worker-a",
  WORKER_A.x + WORKER_W / 2,
  WORKER_A.y + WORKER_H / 2,
  [
    [ARRIVE_A, "busy"],
    [CRASH, "crashed"],
    [RESET, "idle"],
  ],
);

const WORKER_B_TRACK = stateTrack(
  "de-worker-b",
  WORKER_B.x + WORKER_W / 2,
  WORKER_B.y + WORKER_H / 2,
  [
    [ARRIVE_B, "busy"],
    [STEP4_DONE + 750, "idle"],
  ],
);

// ─── Beat annotations (subtitle strip) ─────────────────────────────────────

const beatTrack = (id: string, showAt: number, hideAt: number): FlowTrack =>
  defineTrack(id, [
    { t: showAt, x: BEAT_X, y: BEAT_Y, opacity: 0 },
    { t: showAt + 250, x: BEAT_X, y: BEAT_Y, opacity: 1, ease: "linear" },
    { t: hideAt, x: BEAT_X, y: BEAT_Y, ease: "hold" },
    { t: hideAt + 250, x: BEAT_X, y: BEAT_Y, opacity: 0, ease: "linear" },
  ]);

const BEAT_CRASH = beatTrack("de-beat-crash", CRASH + 150, 6900);
const BEAT_REPLAY = beatTrack("de-beat-replay", 8000, 9200);
const BEAT_LIVE = beatTrack("de-beat-live", 9550, 10700);

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
  <svg viewBox={`0 0 ${STAGE_W} ${STAGE_H}`} aria-hidden="true">
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
    {/* Durable event log rows (empty ledger slots until checkpoints land) */}
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
    {/* Divider between the log and the queue strip */}
    <line
      x1={16}
      y1={DIVIDER_Y}
      x2={128}
      y2={DIVIDER_Y}
      className={styles.chromeDash}
      {...fine}
    />
    {/* Queue strip + dispatch connector toward the workers */}
    <rect
      x={18}
      y={QUEUE_Y - 2}
      width={4}
      height={4}
      className={styles.chromeFill}
    />
    <line
      x1={26}
      y1={QUEUE_Y}
      x2={128}
      y2={QUEUE_Y}
      className={styles.chromeDash}
      {...fine}
    />
    <line
      x1={140}
      y1={QUEUE_Y}
      x2={190}
      y2={QUEUE_Y}
      className={styles.chromeDash}
      {...fine}
    />
    {/* Step cell lanes (worker frames themselves are state-driven tokens) */}
    {[A_CELL_Y, B_CELL_Y].map((y) =>
      CELL_XS.map((x) => (
        <rect
          key={`${x}-${y}`}
          x={x - CELL_W / 2}
          y={y - CELL_W / 2}
          width={CELL_W}
          height={CELL_W}
          className={styles.chromeCell}
          {...fine}
          strokeDasharray="2 3"
        />
      )),
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

const RECORD_LABELS = [
  "step 1 · done",
  "step 2 · done",
  "step 3 · done",
  "step 4 · done",
];

const ARIA_LABEL =
  "Animated diagram of durable execution: a durable task runs step by step on a worker, and each completed step writes a checkpoint into the Hatchet engine's durable event log. Mid-run the worker crashes and the in-flight step is lost, but the log persists. The engine re-dispatches the task to a second worker, which quickly replays steps 1 and 2 from the log without re-executing them, then continues live from step 3 and completes, checkpointing the remaining steps.";

interface DurableExecutionProps {
  className?: string;
  style?: CSSProperties;
}

export const DurableExecution = ({
  className,
  style,
}: DurableExecutionProps) => (
  <div className={`${styles.wrap} ${className ?? ""}`} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.flowRoot}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={10} y={24} anchor="left">
          hatchet engine
        </StageLabel>
        <StageLabel x={18} y={46} anchor="left" muted>
          durable event log
        </StageLabel>
        <StageLabel x={18} y={150} anchor="left" muted>
          queue
        </StageLabel>
        <StageLabel x={WORKER_A.x + 6} y={42} anchor="left" muted>
          worker-a
        </StageLabel>
        <StageLabel x={WORKER_B.x + 6} y={120} anchor="left" muted>
          worker-b
        </StageLabel>
        <Flow.Token track={WORKER_A_TRACK}>
          <div className={styles.workerFrame} />
        </Flow.Token>
        <Flow.Token track={WORKER_B_TRACK}>
          <div className={styles.workerFrame} />
        </Flow.Token>
        {[...A_CELL_TRACKS, ...B_CELL_TRACKS].map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.cellFill} />
          </Flow.Token>
        ))}
        {CHIP_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.chip} />
          </Flow.Token>
        ))}
        {RECORD_TRACKS.map((track, i) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.record}>{RECORD_LABELS[i]}</div>
          </Flow.Token>
        ))}
        <Flow.Token track={BEAT_CRASH}>
          <div className={`${styles.beat} ${styles.beatCrash}`}>
            worker crashed · log persists
          </div>
        </Flow.Token>
        <Flow.Token track={BEAT_REPLAY}>
          <div className={styles.beat}>replay: steps 1–2 skipped</div>
        </Flow.Token>
        <Flow.Token track={BEAT_LIVE}>
          <div className={`${styles.beat} ${styles.beatLive}`}>
            live from step 3
          </div>
        </Flow.Token>
        <Flow.Token track={RUN_1}>
          <div className={styles.square} />
        </Flow.Token>
        <Flow.Token track={RUN_2}>
          <div className={styles.square} />
        </Flow.Token>
      </Flow.Stage>
    </Flow.Root>
  </div>
);
