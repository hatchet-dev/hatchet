"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  createSeededRandom,
  defineTrack,
  resolveTrack,
  sampleTrack,
  type FlowKeyframe,
  type FlowTrack,
  type ResolvedFlowTrack,
} from "@/components/flow";
import styles from "./fairqueueing.module.css";

/**
 * Flow animation for the "Fair queueing" section of the multi-tenant queues
 * post: the resolution to the head-of-line blocking diagram two sections
 * earlier. One mixed stream of tasks fans out at a router into three
 * per-tenant sub-queues — purple, orange, green — and a single worker pops
 * them round-robin: one from purple, one from orange, one from green, forever.
 *
 * The fairness argument is carried by the asymmetry between the lanes. Purple
 * is the noisy tenant: it arrives in bursts and sits on a standing backlog of
 * five or six tasks, while orange and green never hold more than two. Yet the
 * selector cursor visits every lane exactly once per rotation, so purple's
 * depth buys it nothing — the served tape along the top replays the same
 * purple → orange → green cadence no matter how deep purple gets. That is the
 * whole point of round-robin: queue depth cannot buy throughput.
 *
 * The loop is a genuine steady state, not a reset. Each lane takes in exactly
 * as many tasks per loop as it is popped, and tasks that outlive the loop
 * boundary (purple's do — it takes a full rotation-of-rotations to reach the
 * head) are emitted as several time-shifted slices of one logical track, so
 * the standing backlog visible at t = 0 is literally the tail of the tasks
 * that entered during the previous iteration. Everything is scheduled at
 * module scope from plain data; the only variation is seeded.
 */

// ─── Geometry (stage design units, 320 × 200 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 200;

/** Incoming stream: source port → router, along the middle axis. */
const SOURCE_X = 6;
const INLET_Y = 122;
const ROUTER_X = 74;
const ENTRY_X = 94;

/** Sub-queue rails: tasks queue leftward from the head slot. */
const RAIL_X0 = 90;
const HEAD_X = 214;
const SLOT_PITCH = 17;
const slotX = (slot: number) => HEAD_X - slot * SLOT_PITCH;

/** Served tape along the top — the last nine pops, newest on the right. */
const TAPE_Y = 32;
const TAPE_N = 9;
const TAPE_PITCH = 15;
const tapeX = (age: number) => HEAD_X - age * TAPE_PITCH;

/** The worker: one consumer for all three sub-queues. */
const WORKER_X = 283;
const WORKER_Y = INLET_Y;

const TENANTS = ["purple", "orange", "green"] as const;
type Tenant = (typeof TENANTS)[number];

const LANE_Y: Record<Tenant, number> = { purple: 76, orange: 122, green: 168 };

// ─── Timing (ms) ───────────────────────────────────────────────────────────

/** One pop every 500ms, cycling purple → orange → green: a 1500ms rotation. */
const POP_START = 200;
const POP_PITCH = 500;
const ROTATION = POP_PITCH * TENANTS.length;
/** Four full rotations per loop — twelve pops, four per tenant. */
const DURATION = ROTATION * 4;

/** Pop `i` of the whole worker, extended in both directions off the loop. */
const popTime = (i: number) => POP_START + i * POP_PITCH;
const popTenant = (i: number) =>
  TENANTS[((i % TENANTS.length) + TENANTS.length) % TENANTS.length];

const ARRIVE_LEAD = 1400; // source → sub-queue slot
const ADVANCE_MS = 220; // one slot forward when the task ahead is popped
const POP_TRAVEL = 260; // head slot → worker
const POP_DWELL = 200; // held inside the worker
const POP_FADE = 160; // consumed
const CURSOR_SNAP = 120; // selector step onto the next lane…
const CURSOR_LEAD = 80; // …finishing just before that lane is popped
const TAPE_SLIDE = 180; // tape advances one slot per pop

/**
 * Per-tenant arrivals, in loop time. Each lane takes in exactly four tasks per
 * loop and is popped exactly four times, so the composition is periodic.
 * `backlog` is how many pops of that lane a task waits through before it is
 * served — the standing queue depth. Purple's five is what makes it look
 * congested; orange and green stay two deep and are never starved.
 */
interface LaneSpec {
  tenant: Tenant;
  /** Loop time at which each task settles into the back of the sub-queue. */
  arrivals: number[];
  backlog: number;
}

// Arrival times are also spaced so no two tasks overlap on the shared inbound
// run — purple's pairs are the only deliberate bunching.
const LANES: readonly LaneSpec[] = [
  // Bursty and deep: two tasks land back to back, twice per loop.
  { tenant: "purple", arrivals: [500, 800, 3500, 3800], backlog: 5 },
  { tenant: "orange", arrivals: [1000, 2350, 4000, 5350], backlog: 2 },
  { tenant: "green", arrivals: [1350, 2850, 4350, 5850], backlog: 2 },
];

/**
 * Static frame: the beat after purple's third pop. The cursor is parked on
 * purple's head slot, the task it selected is lit inside the worker, two more
 * purple tasks are crossing the router behind it, purple is five deep against
 * two-deep orange and green, and the served tape is full — three complete
 * purple → orange → green rotations. Everything the paragraph claims is
 * legible without a single frame of motion.
 */
const POSTER_TIME = 3500;

// ─── Loop wrapping ─────────────────────────────────────────────────────────

/**
 * Tasks routinely outlive the loop (purple waits ~7s to reach the head), and
 * the queues are pre-filled at t = 0 by the previous iteration's arrivals.
 * Both fall out of the same trick: build each track once in an unbounded time
 * domain, then emit the slices of it that intersect [0, DURATION] under every
 * whole-loop shift. Boundary keyframes are sampled from the track itself, so
 * a slice picks up exactly where the neighbouring slice left off — which is
 * what makes t = DURATION identical to t = 0. Segments that can straddle a
 * boundary are linear or hold, the two easings a re-sampled cut reproduces
 * exactly.
 */
const SHIFTS = [-3, -2, -1, 0, 1].map((n) => n * DURATION);

const boundaryKeyframe = (
  resolved: ResolvedFlowTrack,
  t: number,
): FlowKeyframe => {
  const s = sampleTrack(resolved, t);
  return {
    t,
    x: s.x,
    y: s.y,
    opacity: s.opacity,
    scale: s.scale,
    state: s.state,
    ease: "linear",
  };
};

const clipToLoop = (
  id: string,
  keyframes: FlowKeyframe[],
): FlowTrack | null => {
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

/** Every visible slice of one logically unbounded track. */
const loopTracks = (id: string, keyframes: FlowKeyframe[]): FlowTrack[] =>
  SHIFTS.map((shift) =>
    clipToLoop(
      `${id}~${shift}`,
      keyframes.map((k) => ({ ...k, t: k.t + shift })),
    ),
  ).filter((track): track is FlowTrack => track !== null);

// ─── Task tracks (source → router → sub-queue → worker) ────────────────────

interface TaskTrack {
  tenant: Tenant;
  track: FlowTrack;
}

/** Loop time at which lane `laneIndex` is popped for the `nth` time. */
const lanePopTime = (laneIndex: number, nth: number) =>
  popTime(TENANTS.length * nth + laneIndex);

const rng = createSeededRandom("fair-queueing");

const buildTaskTracks = (): TaskTrack[] =>
  LANES.flatMap(({ tenant, arrivals, backlog }, laneIndex) => {
    const laneY = LANE_Y[tenant];
    return arrivals.flatMap((arrive, j) => {
      const depart = lanePopTime(laneIndex, j + backlog);

      // Every pop of this lane between landing and being served moves the task
      // one slot closer to the head — the queue draining in front of it.
      const advances: number[] = [];
      for (let nth = 0; nth < j + backlog; nth++) {
        const t = lanePopTime(laneIndex, nth);
        if (t > arrive) advances.push(t);
      }

      // Seeded variation in the inbound legs so the stream doesn't march.
      const toRouter = 880 + Math.round(rng() * 80);
      const toEntry = toRouter + 230 + Math.round(rng() * 50);
      const spawn = arrive - ARRIVE_LEAD;

      let slot = advances.length;
      const keyframes: FlowKeyframe[] = [
        { t: spawn, x: SOURCE_X, y: INLET_Y, opacity: 0, state: "inflight" },
        {
          t: spawn + 140,
          x: SOURCE_X + 10,
          y: INLET_Y,
          opacity: 1,
          ease: "linear",
        },
        { t: spawn + toRouter, x: ROUTER_X, y: INLET_Y, ease: "linear" },
        { t: spawn + toEntry, x: ENTRY_X, y: laneY, ease: "linear" },
        {
          t: arrive,
          x: slotX(slot),
          y: laneY,
          ease: "linear",
          state: "queued",
        },
      ];
      for (const t of advances) {
        keyframes.push({ t, x: slotX(slot), y: laneY, ease: "hold" });
        slot -= 1;
        keyframes.push({
          t: t + ADVANCE_MS,
          x: slotX(slot),
          y: laneY,
          ease: "linear",
        });
      }
      keyframes.push(
        { t: depart, x: HEAD_X, y: laneY, ease: "hold", state: "serving" },
        { t: depart + POP_TRAVEL, x: WORKER_X, y: WORKER_Y, ease: "linear" },
        {
          t: depart + POP_TRAVEL + POP_DWELL,
          x: WORKER_X,
          y: WORKER_Y,
          ease: "hold",
        },
        {
          t: depart + POP_TRAVEL + POP_DWELL + POP_FADE,
          x: WORKER_X,
          y: WORKER_Y,
          opacity: 0,
          scale: 0.4,
          ease: "linear",
        },
      );

      return loopTracks(`fq-${tenant}-${j}`, keyframes).map((track) => ({
        tenant,
        track,
      }));
    });
  });

const TASK_TRACKS = buildTaskTracks();

// ─── Served tape (one chip per pop, sliding left) ──────────────────────────

const buildTapeTracks = (): TaskTrack[] =>
  Array.from({ length: DURATION / POP_PITCH }, (_, i) => i).flatMap((i) => {
    const at = popTime(i);
    const keyframes: FlowKeyframe[] = [
      {
        t: at,
        x: tapeX(0),
        y: TAPE_Y,
        opacity: 0,
        scale: 0.5,
        state: "served",
      },
      {
        t: at + 200,
        x: tapeX(0),
        y: TAPE_Y,
        opacity: 1,
        scale: 1,
        ease: "linear",
      },
    ];
    for (let age = 1; age < TAPE_N; age++) {
      keyframes.push({
        t: at + age * POP_PITCH,
        x: tapeX(age - 1),
        y: TAPE_Y,
        ease: "hold",
      });
      keyframes.push({
        t: at + age * POP_PITCH + TAPE_SLIDE,
        x: tapeX(age),
        y: TAPE_Y,
        ease: "linear",
      });
    }
    // Scrolls off the oldest end exactly as the next pop lands on the newest.
    keyframes.push(
      {
        t: at + TAPE_N * POP_PITCH,
        x: tapeX(TAPE_N - 1),
        y: TAPE_Y,
        ease: "hold",
      },
      {
        t: at + TAPE_N * POP_PITCH + TAPE_SLIDE,
        x: tapeX(TAPE_N),
        y: TAPE_Y,
        opacity: 0,
        ease: "linear",
      },
    );
    return loopTracks(`fq-tape-${i}`, keyframes).map((track) => ({
      tenant: popTenant(i),
      track,
    }));
  });

const TAPE_TRACKS = buildTapeTracks();

// ─── Selector cursor + worker ──────────────────────────────────────────────

/**
 * The round-robin cursor: a hollow box that steps down the head slots, landing
 * on a lane a beat before that lane is popped. Its rhythm is the diagram's
 * metronome — purple, orange, green, purple, regardless of how deep any lane
 * has grown.
 */
const CURSOR_TRACK = clipToLoop(
  "fq-cursor",
  Array.from({ length: DURATION / POP_PITCH + 4 }, (_, n) => n - 2).flatMap(
    (i): FlowKeyframe[] => [
      {
        t: popTime(i) - CURSOR_SNAP - CURSOR_LEAD,
        x: HEAD_X,
        y: LANE_Y[popTenant(i - 1)],
        ease: "hold",
      },
      {
        t: popTime(i) - CURSOR_LEAD,
        x: HEAD_X,
        y: LANE_Y[popTenant(i)],
        ease: "linear",
      },
    ],
  ),
);

/** The worker lights up for each pop it absorbs — twelve beats per loop. */
const WORKER_TRACK = clipToLoop(
  "fq-worker",
  Array.from({ length: DURATION / POP_PITCH + 4 }, (_, n) => n - 2).flatMap(
    (i): FlowKeyframe[] => [
      {
        t: popTime(i) + POP_TRAVEL,
        x: WORKER_X,
        y: WORKER_Y,
        ease: "hold",
        state: "busy",
      },
      {
        t: popTime(i) + POP_TRAVEL + POP_DWELL + POP_FADE,
        x: WORKER_X,
        y: WORKER_Y,
        ease: "hold",
        state: "idle",
      },
    ],
  ),
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
    {/* Source port + the single mixed inbound stream */}
    <rect
      x={SOURCE_X - 2}
      y={INLET_Y - 2}
      width={4}
      height={4}
      className={styles.chromeFill}
    />
    <line
      x1={SOURCE_X + 4}
      y1={INLET_Y}
      x2={ROUTER_X - 8}
      y2={INLET_Y}
      className={styles.chromeDash}
      {...fine}
    />
    {/* Router: splits the stream by tenant. A diamond, so it never reads as a
        second selector cursor. */}
    <path
      d={`M ${ROUTER_X} ${INLET_Y - 10} L ${ROUTER_X + 10} ${INLET_Y} L ${ROUTER_X} ${INLET_Y + 10} L ${ROUTER_X - 10} ${INLET_Y} Z`}
      className={styles.chromeStroke}
      {...stroke}
    />
    {/* Fan-out into the three sub-queues */}
    {TENANTS.map((tenant) => (
      <path
        key={`fan-${tenant}`}
        d={`M ${ROUTER_X + 6} ${INLET_Y} C ${ENTRY_X - 6} ${INLET_Y}, ${ROUTER_X + 12} ${LANE_Y[tenant]}, ${ENTRY_X} ${LANE_Y[tenant]}`}
        className={styles.chromeDash}
        {...fine}
      />
    ))}
    {/* Sub-queue rails, head gate, and the stub into the worker */}
    {TENANTS.map((tenant) => (
      <g key={`rail-${tenant}`}>
        <line
          x1={RAIL_X0}
          y1={LANE_Y[tenant]}
          x2={HEAD_X + 10}
          y2={LANE_Y[tenant]}
          className={styles.chromeRail}
          {...fine}
        />
        <line
          x1={HEAD_X + 12}
          y1={LANE_Y[tenant]}
          x2={WORKER_X - 25}
          y2={WORKER_Y}
          className={styles.chromeDash}
          {...fine}
        />
      </g>
    ))}
    {/* The rotation the cursor walks */}
    <line
      x1={HEAD_X}
      y1={LANE_Y.purple - 14}
      x2={HEAD_X}
      y2={LANE_Y.green + 14}
      className={styles.chromeRail}
      {...fine}
    />
  </svg>
);

const StageLabel = ({
  x,
  y,
  anchor = "center",
  muted = false,
  tenant,
  children,
}: {
  x: number;
  y: number;
  anchor?: "center" | "left";
  muted?: boolean;
  tenant?: Tenant;
  children: ReactNode;
}) => (
  <div
    className={[
      styles.stageLabel,
      anchor === "left" ? styles.anchorLeft : "",
      muted ? styles.labelMuted : "",
      tenant ? `${styles.laneLabel} ${styles[tenant]}` : "",
    ]
      .filter(Boolean)
      .join(" ")}
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
  "Animated diagram of round-robin fair queueing: one mixed stream of tasks is split by a router into three per-tenant sub-queues labelled purple, orange and green. Purple arrives in bursts and stays five or six tasks deep while orange and green stay two deep, but a single worker pops one task from each sub-queue in turn, so the served tape along the top repeats purple, orange, green — purple's backlog never starves the other tenants.";

export const FairQueueing = ({ style }: { style?: CSSProperties }) => (
  <div className={styles.wrap} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.flowRoot}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={8} y={TAPE_Y - 5} anchor="left" muted>
          served order
        </StageLabel>
        <StageLabel x={SOURCE_X + 2} y={INLET_Y - 20} anchor="left" muted>
          incoming tasks
        </StageLabel>
        {TENANTS.map((tenant) => (
          <StageLabel
            key={tenant}
            x={ENTRY_X}
            y={LANE_Y[tenant] - 17}
            anchor="left"
            tenant={tenant}
          >
            {tenant}
          </StageLabel>
        ))}
        <StageLabel x={HEAD_X} y={LANE_Y.green + 20} muted>
          round-robin
        </StageLabel>
        <StageLabel x={WORKER_X} y={WORKER_Y - 42}>
          worker
        </StageLabel>
        {WORKER_TRACK && (
          <Flow.Token track={WORKER_TRACK}>
            <div className={styles.workerBox} />
          </Flow.Token>
        )}
        {CURSOR_TRACK && (
          <Flow.Token track={CURSOR_TRACK}>
            <div className={styles.cursor} />
          </Flow.Token>
        )}
        {TAPE_TRACKS.map(({ tenant, track }) => (
          <Flow.Token key={track.id} track={track} className={styles[tenant]}>
            <div className={styles.chip} />
          </Flow.Token>
        ))}
        {TASK_TRACKS.map(({ tenant, track }) => (
          <Flow.Token key={track.id} track={track} className={styles[tenant]}>
            <div className={styles.square} />
          </Flow.Token>
        ))}
      </Flow.Stage>
    </Flow.Root>
  </div>
);
