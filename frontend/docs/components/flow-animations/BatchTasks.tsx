"use client";

import type { CSSProperties, ReactNode } from "react";
import { Flow, defineTrack, type FlowKeyframe, type FlowTrack } from "@/components/flow";
import { Text } from "@/components/flow/Text";
import styles from "./batchtasks.module.css";

/**
 * Flow animation of Hatchet batch tasks (docs.hatchet.run/v1/batch-tasks):
 * individual task triggers from three callers accumulate in a batch buffer
 * instead of executing immediately. The buffer flushes when EITHER it reaches
 * max size OR the interval timer runs out — beat 1 fills all three slots and
 * flushes on size (the tick-bar barely starts depleting); beat 2 buffers only
 * two tasks and the tick-bar runs all the way down, flushing a partial batch
 * on time. Each flush fuses the buffered squares into one bracketed batch
 * token that a single handler invocation processes (accent blue), after which
 * each caller receives back only the output for its own input — the results
 * fan back out to the originating lanes, so batching stays invisible to
 * callers.
 *
 * All timings are hand-scheduled at module scope, so the loop is fully
 * deterministic and the composition at t = DURATION matches t = 0.
 */

// ─── Geometry (stage design units, 320 × 200 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 200;

/** Caller lanes: three ports on the left edge. */
const LANE = { a: 64, b: 108, c: 152 } as const;
type LaneKey = keyof typeof LANE;
const LANES: readonly LaneKey[] = ["a", "b", "c"];
const PORT_X = 14;

/** Batch buffer: vertical bracketed strip with 3 capacity slots. */
const BUF_X = 150;
const SLOT_YS = [93, 108, 123] as const;
const BUF_TOP = 80;
const BUF_BOTTOM = 136;

/** Interval indicator: a depleting bar of 8 ticks under the buffer. */
const TICKS = 8;
const TICK_Y = 146;
const tickX = (j: number) => 136 + j * 4;

/** Handler box on the right. */
const WORKER_X = 262;
const WORKER_Y = 108;
const WORKER_H = 32;

/** Return bus: results drop out of the handler and travel back along it. */
const BUS_Y = 176;
/** Where each lane's result leaves the bus and rises to its port. */
const TURN_X: Record<LaneKey, number> = { a: 60, b: 78, c: 96 };

// ─── Timing (ms) — the docs' "3 tasks or 200ms" stretched for legibility ───

const TRAVEL_IN = 800; // caller port → buffer slot
const INTERVAL = 3000; // interval timer: arm → forced flush
const TICK_MS = INTERVAL / TICKS;
const FUSE_MS = 350; // buffered squares converge into the batch stack
const DEPART_GAP = 500; // flush → batch leaves the buffer
const DISPATCH_MS = 600; // buffer → handler
const PROC_MS = 1200; // one handler invocation for the whole batch
const RET_STAGGER = 140; // results leave the handler slightly staggered
const DROP_MS = 250; // handler → return bus
const BUS_SPEED = 2.8; // ms per design unit along the bus
const RISE_MS = 420; // bus → caller port
const DWELL_MS = 300; // delivered blink at the port
const FADE_MS = 300;

interface TaskSpec {
  lane: LaneKey;
  spawn: number;
}

interface Beat {
  id: string;
  tasks: TaskSpec[];
  /** Interval timer arms when the first task lands in the empty buffer. */
  arm: number;
  /** Buffer flushes: beat 1 on max size, beat 2 on interval expiry. */
  flush: number;
  depart: number;
  arrive: number;
  procEnd: number;
}

const makeBeat = (id: string, tasks: TaskSpec[], flush: number): Beat => {
  const depart = flush + DEPART_GAP;
  const arrive = depart + DISPATCH_MS;
  return {
    id,
    tasks,
    arm: tasks[0].spawn + TRAVEL_IN,
    flush,
    depart,
    arrive,
    procEnd: arrive + PROC_MS,
  };
};

/** Beat 1 — three tasks arrive quickly; the buffer flushes on max size. */
const BEAT1 = makeBeat(
  "b1",
  [
    { lane: "a", spawn: 300 },
    { lane: "c", spawn: 700 },
    { lane: "b", spawn: 1050 },
  ],
  1050 + TRAVEL_IN + 200 // shortly after the third task lands: size wins
);

/** Beat 2 — only two tasks arrive; the interval timer forces a partial flush. */
const BEAT2 = makeBeat(
  "b2",
  [
    { lane: "b", spawn: 6800 },
    { lane: "c", spawn: 7900 },
  ],
  6800 + TRAVEL_IN + INTERVAL // exactly when the tick-bar runs out
);

const BEATS = [BEAT1, BEAT2];

const DURATION = 15200;
/** Static frame: the fused batch mid-processing inside the blue handler. */
const POSTER_TIME = 3800;

// ─── Track builders ────────────────────────────────────────────────────────

/** Fused batch stack: n squares centered on the handler axis, pitch 6. */
const stackY = (n: number, i: number) => WORKER_Y + (i - (n - 1) / 2) * 6;

/**
 * One token per task, round-tripping: spawn at its caller port → buffer slot
 * → fuse into the batch stack → ride to the handler (accent while the single
 * invocation runs) → morph into a hollow result square → drop to the return
 * bus and fan back out to its own originating lane.
 */
const taskTrack = (beat: Beat, i: number): FlowTrack => {
  const { lane, spawn } = beat.tasks[i];
  const laneY = LANE[lane];
  const slotY = SLOT_YS[i];
  const stack = stackY(beat.tasks.length, i);
  const retDepart = beat.procEnd + 100 + i * RET_STAGGER;
  const busEnd = retDepart + DROP_MS + Math.round((WORKER_X - TURN_X[lane]) * BUS_SPEED);
  const portAt = busEnd + RISE_MS;
  return defineTrack(`batch-${beat.id}-task-${i}`, [
    { t: spawn, x: 10, y: laneY, opacity: 0, state: "inflight" },
    { t: spawn + 180, x: 26, y: laneY, opacity: 1, ease: "linear" },
    { t: spawn + 560, x: 112, y: laneY, ease: "linear" },
    { t: spawn + TRAVEL_IN, x: BUF_X, y: slotY, ease: "out", state: "buffered" },
    { t: beat.flush, x: BUF_X, y: slotY, ease: "hold" },
    { t: beat.flush + FUSE_MS, x: BUF_X, y: stack, ease: "inOut", state: "batched" },
    { t: beat.depart, x: BUF_X, y: stack, ease: "hold" },
    { t: beat.arrive, x: WORKER_X, y: stack, ease: "inOut", state: "processing" },
    { t: beat.procEnd, x: WORKER_X, y: stack, ease: "hold", state: "result" },
    { t: retDepart, x: WORKER_X, y: stack, ease: "hold" },
    { t: retDepart + DROP_MS, x: WORKER_X, y: BUS_Y, ease: "in" },
    { t: busEnd, x: TURN_X[lane], y: BUS_Y, ease: "linear" },
    { t: portAt, x: PORT_X, y: laneY, ease: "out", state: "delivered" },
    { t: portAt + DWELL_MS, x: PORT_X, y: laneY, ease: "hold" },
    { t: portAt + DWELL_MS + FADE_MS, x: PORT_X, y: laneY, opacity: 0, ease: "linear" },
  ]);
};

const TASK_TRACKS = BEATS.flatMap((beat) => beat.tasks.map((_, i) => taskTrack(beat, i)));

/** The bracket that fuses around a flushing batch and rides to the handler. */
const bracketTrack = (beat: Beat): FlowTrack =>
  defineTrack(`batch-${beat.id}-bracket`, [
    { t: beat.flush, x: BUF_X, y: WORKER_Y, opacity: 0 },
    { t: beat.flush + 250, x: BUF_X, y: WORKER_Y, opacity: 1, ease: "linear" },
    { t: beat.depart, x: BUF_X, y: WORKER_Y, ease: "hold" },
    { t: beat.arrive, x: WORKER_X, y: WORKER_Y, ease: "inOut" },
    { t: beat.procEnd, x: WORKER_X, y: WORKER_Y, ease: "hold" },
    { t: beat.procEnd + 220, x: WORKER_X, y: WORKER_Y, opacity: 0, ease: "linear" },
  ]);

const BRACKET_TRACKS = BEATS.map(bracketTrack);

/**
 * Interval tick j: dim while the buffer is empty, bright once the timer arms,
 * then the bar depletes right-to-left — one tick per TICK_MS. Beat 1's size
 * flush refills it after only two ticks burn; beat 2 burns all eight, and the
 * last tick dies exactly at the flush.
 */
const tickTrack = (j: number): FlowTrack => {
  const x = tickX(j);
  const kfs: FlowKeyframe[] = [{ t: 0, x, y: TICK_Y, state: "idle" }];
  for (const beat of BEATS) {
    kfs.push({ t: beat.arm, x, y: TICK_Y, ease: "hold", state: "armed" });
    const vanish = beat.arm + (TICKS - j) * TICK_MS;
    if (vanish <= beat.flush) {
      kfs.push(
        { t: vanish, x, y: TICK_Y, ease: "hold" },
        { t: vanish + 140, x, y: TICK_Y, opacity: 0, ease: "linear" }
      );
    }
    kfs.push({ t: beat.flush + 200, x, y: TICK_Y, ease: "hold", state: "idle" });
    kfs.push({ t: beat.flush + 400, x, y: TICK_Y, opacity: 1, ease: "linear" });
  }
  kfs.push({ t: DURATION, x, y: TICK_Y, ease: "hold" });
  return defineTrack(`batch-tick-${j}`, kfs);
};

const TICK_TRACKS = Array.from({ length: TICKS }, (_, j) => tickTrack(j));

/** The handler box brightens for exactly one invocation per batch. */
const WORKER_TRACK = defineTrack("batch-worker", [
  { t: 0, x: WORKER_X, y: WORKER_Y, state: "idle" },
  ...BEATS.flatMap((beat): FlowKeyframe[] => [
    { t: beat.arrive, x: WORKER_X, y: WORKER_Y, state: "busy", ease: "hold" },
    { t: beat.procEnd, x: WORKER_X, y: WORKER_Y, state: "idle", ease: "hold" },
  ]),
  { t: DURATION, x: WORKER_X, y: WORKER_Y, ease: "hold" },
]);

// ─── Static chrome ─────────────────────────────────────────────────────────

const stroke = { fill: "none", strokeWidth: 1.5, vectorEffect: "non-scaling-stroke" } as const;
const fine = { fill: "none", strokeWidth: 1, vectorEffect: "non-scaling-stroke" } as const;

const Chrome = () => (
  <svg viewBox={`0 0 ${STAGE_W} ${STAGE_H}`} aria-hidden="true" className={styles.chrome}>
    {/* Caller ports + dashed lanes toward the buffer */}
    {LANES.map((lane) => (
      <g key={lane}>
        <rect x={8} y={LANE[lane] - 2} width={4} height={4} className={styles.chromeFill} />
        <line
          x1={16}
          y1={LANE[lane]}
          x2={112}
          y2={LANE[lane]}
          className={styles.chromeDash}
          {...fine}
        />
      </g>
    ))}
    {/* Buffer brackets (top + bottom of the accumulator strip) */}
    <path
      d={`M ${BUF_X - 7} ${BUF_TOP + 6} L ${BUF_X - 7} ${BUF_TOP} L ${BUF_X + 7} ${BUF_TOP} L ${BUF_X + 7} ${BUF_TOP + 6}`}
      className={styles.chromeStroke}
      {...stroke}
    />
    <path
      d={`M ${BUF_X - 7} ${BUF_BOTTOM - 6} L ${BUF_X - 7} ${BUF_BOTTOM} L ${BUF_X + 7} ${BUF_BOTTOM} L ${BUF_X + 7} ${BUF_BOTTOM - 6}`}
      className={styles.chromeStroke}
      {...stroke}
    />
    {/* Capacity ticks — 3 reserved slots */}
    {SLOT_YS.map((y) => (
      <line
        key={y}
        x1={BUF_X - 3.5}
        y1={y}
        x2={BUF_X + 3.5}
        y2={y}
        className={styles.chromeTick}
        {...fine}
      />
    ))}
    {/* Interval tick-bar ghost (dim positions under the live ticks) */}
    {Array.from({ length: TICKS }, (_, j) => (
      <line
        key={j}
        x1={tickX(j)}
        y1={TICK_Y - 2.5}
        x2={tickX(j)}
        y2={TICK_Y + 2.5}
        className={styles.chromeTick}
        {...fine}
      />
    ))}
    {/* Buffer → handler connector */}
    <line x1={162} y1={WORKER_Y} x2={236} y2={WORKER_Y} className={styles.chromeDash} {...fine} />
    {/* Return bus: handler → back toward the callers */}
    <path
      d={`M ${WORKER_X} ${WORKER_Y + WORKER_H / 2 + 4} V ${BUS_Y} H 88`}
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
  "Animated diagram of Hatchet batch tasks: tasks from three callers accumulate in a batch buffer that flushes when it reaches max size or when the interval timer runs out. Each batch executes in a single handler invocation, and each caller receives back only its own result.";

const CAPTION =
  "Tasks accumulate into a batch and flush when max size or interval are reached.";

export const BatchTasks = ({
  style,
  showCaption = true,
}: {
  style?: CSSProperties;
  showCaption?: boolean;
}) => (
  <div className={styles.batchTasks} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.flowRoot}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={8} y={44} anchor="left">
          callers
        </StageLabel>
        <StageLabel x={BUF_X} y={44}>
          batch buffer · max 3
        </StageLabel>
        <StageLabel x={WORKER_X} y={44}>
          one handler
        </StageLabel>
        <StageLabel x={BUF_X} y={154} muted>
          interval
        </StageLabel>
        <StageLabel x={170} y={182} muted>
          results
        </StageLabel>
        <Flow.Token track={WORKER_TRACK}>
          <div className={styles.workerBox} />
        </Flow.Token>
        {BRACKET_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.batchFrame} />
          </Flow.Token>
        ))}
        {TICK_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.tick} />
          </Flow.Token>
        ))}
        {TASK_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.square} />
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
