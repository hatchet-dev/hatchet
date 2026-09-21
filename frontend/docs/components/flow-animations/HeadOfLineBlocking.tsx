"use client";

import { useRef, type CSSProperties, type ReactNode } from "react";
import {
  Flow,
  createSeededRandom,
  defineTrack,
  useFlow,
  useFlowFrame,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import styles from "./headoflineblocking.module.css";

/**
 * Flow animation for the head-of-line blocking beat of the multi-tenant
 * queues post: once you bottleneck the system with a single FIFO queue and a
 * worker that takes one task at a time, whoever gets there first owns the
 * queue.
 *
 * Bob's hard drive floods in first and packs all fifteen slots of the shared
 * queue. Alice's task arrives a moment later — a visibly tiny square, one
 * page against fifteen of Bob's files — and FIFO puts her at the tail, the
 * furthest possible point from the worker. The worker then drains the head
 * one slow task at a time; the whole line shifts right by one slot per
 * service, and Alice creeps forward by a grand total of five places before
 * the loop wraps. The live readout under the queue counts the files still
 * ahead of her: 15 → 10. Nothing failed, nothing is misconfigured — she is
 * simply behind Bob, and FIFO has no opinion about that.
 *
 * All timings are hand-scheduled at module scope (with seeded jitter on the
 * flood), so the loop is fully deterministic and the composition at
 * t = DURATION matches the empty stage at t = 0.
 */

// ─── Geometry (stage design units, 320 × 108 — wide, short, in-article) ────

const STAGE_W = 320;
const STAGE_H = 108;

/** The shared queue: one horizontal channel, head on the right. */
const QUEUE_Y = 58;
const HEAD_X = 236;
const PITCH = 12;
const CHANNEL_X0 = 48;
const CHANNEL_X1 = 244;
const RAIL_TOP = 49;
const RAIL_BOTTOM = 67;

/** Slot 0 is the head (served next); the index is also "how many are ahead". */
const slotX = (slot: number) => HEAD_X - slot * PITCH;

/** Bob fills every slot but the last; Alice lands in the tail slot. */
const BOB_N = 15;
const ALICE_SLOT = BOB_N; // → x = 56, hard against the tail rail

/** Tenant ports on the left edge, each with a dashed approach into the tail. */
const BOB_PORT = { x: 12, y: 28 };
const ALICE_PORT = { x: 12, y: 82 };

/** The single worker: one slot, one task at a time. */
const WORKER_X = 290;
const READOUT_X = 147;
const READOUT_Y = 92;

// ─── Timing (ms) ───────────────────────────────────────────────────────────

const FLOOD_START = 150; // Bob's first file
const FLOOD_GAP = 105; // he is uploading a hard drive, not sending requests
const LANE_MS = 380; // port → the mouth of the channel
const CHANNEL_MS_PER_U = 3; // constant speed down the channel to a free slot
const SETTLE_MS = 60;

const ALICE_SPAWN = 2150; // one page, arriving just after the flood lands
const ALICE_SETTLED = 3050;

const DRAIN_START = 3400; // the worker starts pulling from the head
const CYCLE = 800; // one service: Bob's files are big
const DRAINS = 5; // …and only five of them clear in a whole loop
const SHIFT_MS = 340; // the line closes up behind each departure
const TO_WORKER_MS = 320;
const PROC_MS = 380;

const READOUT_IN = 3250;
const FADE_START = 7850; // hold on the stranded frame, then reset
const FADE_END = 8200;
const DURATION = 8400;

/** Loop time at which the worker pulls service `j`. */
const cycleAt = (j: number) => DRAIN_START + j * CYCLE;

/**
 * Static frame: three services in, worker busy, twelve of Bob's files still
 * stacked between Alice and the head. This is the whole argument in one
 * picture, so it is what crawlers and reduced-motion readers get.
 */
const POSTER_TIME = 5450;

// ─── Flood schedule (seeded jitter — deterministic, never Math.random) ─────

const rng = createSeededRandom("head-of-line-blocking");
const BOB_SPAWNS = Array.from(
  { length: BOB_N },
  (_, slot) => FLOOD_START + slot * FLOOD_GAP + Math.round(rng() * 60) - 30,
);

// ─── Track builders ────────────────────────────────────────────────────────

/**
 * Every task walks the same queue geometry once it is parked: at each service
 * the head leaves and everyone behind it shifts one slot to the right. A task
 * that starts in slot `slot` is therefore in slot `slot - j` after service j.
 */
const shiftKeyframes = (slot: number, shifts: number): FlowKeyframe[] =>
  Array.from({ length: shifts }, (_, j) => [
    { t: cycleAt(j), x: slotX(slot - j), y: QUEUE_Y, ease: "hold" as const },
    {
      t: cycleAt(j) + SHIFT_MS,
      x: slotX(slot - j - 1),
      y: QUEUE_Y,
      ease: "inOut" as const,
    },
  ]).flat();

/** Tail of a task that never gets served inside the loop: hold, then reset. */
const strandedKeyframes = (restX: number): FlowKeyframe[] => [
  { t: FADE_START, x: restX, y: QUEUE_Y, ease: "hold" },
  { t: FADE_END, x: restX, y: QUEUE_Y, opacity: 0, ease: "linear" },
];

/**
 * One of Bob's files: spawn at his port, run down the channel to the first
 * free slot, wait its turn, and — for the lucky first five — get pulled to the
 * worker, processed, and dropped.
 */
const bobTrack = (slot: number): FlowTrack => {
  const spawn = BOB_SPAWNS[slot];
  const enter = spawn + LANE_MS;
  const parked =
    enter +
    Math.round((slotX(slot) - CHANNEL_X0 - 4) * CHANNEL_MS_PER_U) +
    SETTLE_MS;
  const approach: FlowKeyframe[] = [
    { t: spawn, x: BOB_PORT.x, y: BOB_PORT.y, opacity: 0, state: "queued" },
    { t: spawn + 150, x: 28, y: 37, opacity: 1, ease: "linear" },
    { t: enter, x: CHANNEL_X0 + 4, y: QUEUE_Y, ease: "linear" },
    { t: parked, x: slotX(slot), y: QUEUE_Y, ease: "out" },
  ];
  if (slot >= DRAINS) {
    return defineTrack(`hlb-bob-${slot}`, [
      ...approach,
      ...shiftKeyframes(slot, DRAINS),
      ...strandedKeyframes(slotX(slot - DRAINS)),
    ]);
  }
  const served = cycleAt(slot);
  return defineTrack(`hlb-bob-${slot}`, [
    ...approach,
    ...shiftKeyframes(slot, slot),
    { t: served, x: HEAD_X, y: QUEUE_Y, ease: "hold" },
    {
      t: served + TO_WORKER_MS,
      x: WORKER_X,
      y: QUEUE_Y,
      ease: "inOut",
      state: "processing",
    },
    {
      t: served + TO_WORKER_MS + PROC_MS,
      x: WORKER_X,
      y: QUEUE_Y,
      ease: "hold",
      state: "done",
    },
    {
      t: served + CYCLE - 20,
      x: WORKER_X,
      y: QUEUE_Y,
      opacity: 0,
      scale: 1.4,
      ease: "out",
    },
  ]);
};

const BOB_TRACKS = Array.from({ length: BOB_N }, (_, slot) => bobTrack(slot));

/**
 * Alice's one page. Same channel, same rules — she just got there second, so
 * FIFO hands her the tail slot and she never reaches the head in this loop.
 */
const ALICE_TRACK = defineTrack("hlb-alice", [
  {
    t: ALICE_SPAWN,
    x: ALICE_PORT.x,
    y: ALICE_PORT.y,
    opacity: 0,
    state: "waiting",
  },
  { t: ALICE_SPAWN + 150, x: 26, y: 76, opacity: 1, ease: "linear" },
  { t: ALICE_SPAWN + 500, x: 44, y: 64, ease: "linear" },
  { t: ALICE_SETTLED, x: slotX(ALICE_SLOT), y: QUEUE_Y, ease: "out" },
  ...shiftKeyframes(ALICE_SLOT, DRAINS),
  ...strandedKeyframes(slotX(ALICE_SLOT - DRAINS)),
]);

/** The worker box: busy for exactly one task at a time, idle in between. */
const WORKER_TRACK = defineTrack("hlb-worker", [
  { t: 0, x: WORKER_X, y: QUEUE_Y, state: "idle" },
  ...Array.from({ length: DRAINS }, (_, j): FlowKeyframe[] => [
    {
      t: cycleAt(j) + TO_WORKER_MS,
      x: WORKER_X,
      y: QUEUE_Y,
      ease: "hold",
      state: "busy",
    },
    {
      t: cycleAt(j) + TO_WORKER_MS + PROC_MS,
      x: WORKER_X,
      y: QUEUE_Y,
      ease: "hold",
      state: "idle",
    },
  ]).flat(),
  { t: DURATION, x: WORKER_X, y: QUEUE_Y, ease: "hold" },
]);

// ─── Live readout (per-frame text, no React renders) ───────────────────────

const clamp = (v: number, lo: number, hi: number) =>
  Math.min(hi, Math.max(lo, v));
const ramp = (t: number, a: number, b: number) =>
  clamp((t - a) / (b - a), 0, 1);

/** A service has "landed" once the line has finished closing up behind it. */
const servicesLanded = (t: number) => {
  let n = 0;
  for (let j = 0; j < DRAINS; j++) if (t >= cycleAt(j) + SHIFT_MS) n += 1;
  return n;
};

/** Alice's slot index is literally the number of files queued ahead of her. */
const filesAhead = (t: number) => ALICE_SLOT - servicesLanded(t);

const readoutOpacity = (t: number) =>
  ramp(t, READOUT_IN, READOUT_IN + 250) * (1 - ramp(t, FADE_START, FADE_END));

/**
 * The counter that makes the unfairness quantitative: it appears when Alice
 * parks and barely moves for the rest of the loop.
 */
const AheadReadout = () => {
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
    const next = String(filesAhead(t));
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
        {filesAhead(posterTime)}
      </span>{" "}
      of bob&apos;s files ahead of alice
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
    {/* Tenant ports + their dashed approaches into the tail of the queue */}
    {[BOB_PORT, ALICE_PORT].map((port) => (
      <rect
        key={port.y}
        x={port.x - 2}
        y={port.y - 2}
        width={4}
        height={4}
        className={styles.chromeFill}
      />
    ))}
    <line
      x1={18}
      y1={BOB_PORT.y}
      x2={46}
      y2={52}
      className={styles.chromeDash}
      {...fine}
    />
    <line
      x1={18}
      y1={ALICE_PORT.y}
      x2={44}
      y2={64}
      className={styles.chromeDash}
      {...fine}
    />
    {/* The one shared channel — open at the tail, served at the head */}
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
    {/* Head → worker: the single-file exit that everyone is waiting on */}
    <line
      x1={CHANNEL_X1 + 4}
      y1={QUEUE_Y}
      x2={266}
      y2={QUEUE_Y}
      className={styles.chromeDash}
      {...fine}
    />
    <polygon
      points={`270,${QUEUE_Y} 263,${QUEUE_Y - 3.5} 263,${QUEUE_Y + 3.5}`}
      className={styles.chromeFill}
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
  "Animated diagram of head-of-line blocking in a single shared FIFO queue. Bob's upload floods the queue and fills every slot. Alice's one-page upload arrives next and FIFO places her at the tail, furthest from the worker. The worker serves one task at a time from the head, so the line shifts forward slowly and Alice is still stuck behind ten of Bob's files at the end of the loop.";

export const HeadOfLineBlocking = ({ style }: { style?: CSSProperties }) => (
  <div className={styles.wrap} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.flowRoot}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={8} y={12} anchor="left">
          bob · 10,000 files
        </StageLabel>
        <StageLabel x={147} y={12}>
          one fifo queue
        </StageLabel>
        <StageLabel x={WORKER_X} y={12}>
          worker
        </StageLabel>
        <StageLabel x={8} y={90} anchor="left">
          alice
        </StageLabel>
        <StageLabel x={56} y={38} muted>
          tail
        </StageLabel>
        <StageLabel x={HEAD_X} y={38} muted>
          head
        </StageLabel>
        <StageLabel x={WORKER_X} y={80} muted>
          1 at a time
        </StageLabel>
        <Flow.Token track={WORKER_TRACK}>
          <div className={styles.workerBox} />
        </Flow.Token>
        {BOB_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.bobSquare} />
          </Flow.Token>
        ))}
        <Flow.Token track={ALICE_TRACK}>
          <div className={styles.aliceTask}>
            <div className={styles.aliceSquare} />
            <div className={styles.aliceTag}>alice · 1 page</div>
          </div>
        </Flow.Token>
        <AheadReadout />
      </Flow.Stage>
    </Flow.Root>
  </div>
);
