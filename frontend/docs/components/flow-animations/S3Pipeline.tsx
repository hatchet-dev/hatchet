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
import { Text } from "@/components/flow/Text";
import styles from "./s3pipeline.module.css";

/**
 * Flow animation for the S3 processing-pipeline cookbook: the steady state of
 * the fan-out described in the surrounding prose.
 *
 * Beat 1 (the tick) — a cron pulse fires at the port on the left and three
 * hollow `list_objects` child tasks fan out, one per bucket.
 * Beat 2 (the fan-out) — as each `list_objects` task lands, its bucket starts
 * streaming object tokens rightward into a per-bucket queue. bucket-1 holds
 * the most objects, so its queue grows visibly deeper than the others.
 * Beat 3 (the drain) — a single worker with three slots pulls queue heads
 * round-robin: the accent cursor snaps from head to head, and the three slots
 * hold objects from different buckets at once, each with its own progress
 * bar. Depth buys bucket-1 nothing — the rotation visits every bucket.
 * Beat 4 (the tally) — finished objects leave the worker hollow and settle
 * into the processed column on the right, interleaved colours proving the
 * round-robin. When the backlog hits zero the tally fades and the loop wraps
 * to the empty pre-tick state.
 *
 * The schedule is a small round-robin simulation run once at module scope
 * (three worker slots, per-bucket FIFO queues), so every pull time, shuffle
 * and tally position is derived data — identical on the server, on the poster
 * frame, and on every loop iteration. The only variation is seeded jitter on
 * emission gaps and service times so nothing marches in lockstep.
 */

// ─── Geometry (stage design units, 320 × 210 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 210;

/** One lane per bucket; the cron port sits on the middle axis. */
const LANE_Y = [57, 105, 153] as const;
const MID_Y = 105;

const CRON_X = 14;
const BUCKET_X = 64;

/** Per-bucket queue rails: slot 0 is the head; the queue fills leftward. */
const HEAD_X = 176;
const SLOT_PITCH = 12;
const slotX = (depth: number) => HEAD_X - depth * SLOT_PITCH;

/** Worker: one box, three slots, each slot a square + progress bar. */
const SLOTS = 3;
const WORKER_BOX = { x: 204, y: 46, w: 52, h: 118 };
const RUN_X = 216;
const SLOT_Y = [70, 105, 140] as const;
const BAR_X = 226;
const BAR_W = 22;
const BAR_H = 2;

/** Processed objects settle into a colour-interleaved column on the right. */
const TALLY_X = 296;
const TALLY_BASE = 172;
const TALLY_PITCH = 7.5;
const tallyY = (k: number) => TALLY_BASE - k * TALLY_PITCH;

/** Readout row under the composition. */
const READOUT_Y = 186;

// ─── Buckets ───────────────────────────────────────────────────────────────

/** bucket-1 is the deep one — the prose's "most pending objects" bucket. */
const BUCKETS = [
  { name: "bucket-0", objects: 4, emitGap: 230 },
  { name: "bucket-1", objects: 7, emitGap: 170 },
  { name: "bucket-2", objects: 4, emitGap: 230 },
] as const;

// ─── Timing (ms) ───────────────────────────────────────────────────────────

const CRON_AT = 300; // the tick fires
const TRIGGER_TRAVEL = 700; // cron port → bucket, staggered per lane
const TRAVEL_IN = 480; // bucket → tail of its queue
const MIN_WAIT = 120; // an object always lands at the head before departing
const PULL_MS = 380; // queue head → worker slot
const PROC_MS = 900; // base service time per object
const PROC_JITTER = 250;
const SLOT_GAP = 120; // slot frees → next pull begins
const MIN_PULL_GAP = 90; // two slots freeing together still pull one at a time
const SHUFFLE_MS = 150; // queue shuffles one slot toward the head
const DONE_DWELL = 220; // finished object held in its slot, hollow
const EXIT_MS = 420; // worker slot → tally column
const FADE_MS = 350;

const TRIGGER_ARRIVE = BUCKETS.map(
  (_, lane) => CRON_AT + TRIGGER_TRAVEL + lane * 140,
);

const rng = createSeededRandom("s3-pipeline");

// ─── Schedule (round-robin simulation, run once at module scope) ───────────

interface Emission {
  index: number;
  lane: number;
  /** Leaves the bucket. */
  emitAt: number;
  /** Lands at the tail of its bucket's queue. */
  arriveAt: number;
}

const EMISSIONS: Emission[] = (() => {
  const out: Emission[] = [];
  BUCKETS.forEach((bucket, lane) => {
    let t = TRIGGER_ARRIVE[lane] + 200;
    for (let j = 0; j < bucket.objects; j++) {
      out.push({ index: out.length, lane, emitAt: t, arriveAt: t + TRAVEL_IN });
      t += bucket.emitGap + Math.round(rng() * 70);
    }
  });
  return out;
})();

interface ObjectRun extends Emission {
  /** A worker slot frees and this bucket is next in the rotation. */
  pullAt: number;
  /** Processing starts — the object has reached its slot. */
  startAt: number;
  endAt: number;
  procMs: number;
  slot: number;
  /** Queue depth on landing. */
  enterDepth: number;
  /** Pull times of queue-mates ahead: each one shuffles this object right. */
  shuffles: number[];
  /** Completion order — its resting row in the processed column. */
  tally: number;
}

const RUNS: ObjectRun[] = (() => {
  const pending = [...EMISSIONS].sort((a, b) => a.arriveAt - b.arriveAt);
  const queues: Emission[][] = BUCKETS.map(() => []);
  const slotFree = Array.from({ length: SLOTS }, () => 0);
  let rr = 0;
  let p = 0;
  let lastPull = -MIN_PULL_GAP;
  const admit = (t: number) => {
    while (p < pending.length && pending[p].arriveAt <= t) {
      queues[pending[p].lane].push(pending[p]);
      p++;
    }
  };

  const out: ObjectRun[] = [];
  while (out.length < pending.length) {
    // The next slot to free is the next pull opportunity…
    let slot = 0;
    for (let s = 1; s < SLOTS; s++) if (slotFree[s] < slotFree[slot]) slot = s;
    let t = slotFree[slot];
    admit(t);
    // …unless every queue is empty, in which case it idles until an arrival.
    if (queues.every((q) => q.length === 0)) {
      t = pending[p].arriveAt;
      admit(t);
    }
    // Round-robin: the rotation pointer picks the next non-empty bucket.
    let lane = rr;
    while (queues[lane].length === 0) lane = (lane + 1) % BUCKETS.length;
    const obj = queues[lane].shift()!;
    const pullAt = Math.max(
      t,
      obj.arriveAt + MIN_WAIT,
      lastPull + MIN_PULL_GAP,
    );
    lastPull = pullAt;
    const startAt = pullAt + PULL_MS;
    const procMs = PROC_MS + Math.round(rng() * PROC_JITTER);
    const endAt = startAt + procMs;
    slotFree[slot] = endAt + SLOT_GAP;
    rr = (lane + 1) % BUCKETS.length;
    out.push({
      ...obj,
      pullAt,
      startAt,
      endAt,
      procMs,
      slot,
      enterDepth: 0,
      shuffles: [],
      tally: 0,
    });
  }

  // Per-lane FIFO: every queue-mate still waiting when this object lands is
  // pulled before it, and each of those pulls shuffles it one slot forward.
  for (const run of out) {
    run.shuffles = out
      .filter(
        (o) =>
          o.lane === run.lane &&
          o.arriveAt < run.arriveAt &&
          o.pullAt > run.arriveAt + 30,
      )
      .map((o) => o.pullAt)
      .sort((a, b) => a - b);
    run.enterDepth = run.shuffles.length;
  }

  [...out]
    .sort((a, b) => a.endAt - b.endAt)
    .forEach((run, k) => {
      run.tally = k;
    });

  return out;
})();

/** How deep each queue actually gets — its rail is drawn exactly that deep. */
const MAX_DEPTH = BUCKETS.map((_, lane) =>
  Math.max(0, ...RUNS.filter((r) => r.lane === lane).map((r) => r.enterDepth)),
);

const LAST_SETTLE =
  Math.max(...RUNS.map((r) => r.endAt)) + DONE_DWELL + EXIT_MS;
/** The full tally dwells — the batch's poster of completion — then fades. */
const FADE_AT = LAST_SETTLE + 900;
const FADE_END = FADE_AT + FADE_MS;
const DURATION = Math.ceil((FADE_END + 500) / 200) * 200;

const PULLS = [...RUNS].sort((a, b) => a.pullAt - b.pullAt);

/**
 * Static frame: mid-drain. All three worker slots hold objects from different
 * buckets, the cursor is parked on a queue head, bucket-1's queue is visibly
 * the deepest, and a few interleaved chips already sit in the tally column.
 */
const POSTER_TIME = PULLS[5].startAt + 250;

// ─── Object tracks (bucket → queue → worker slot → tally) ──────────────────

const objectTrack = (run: ObjectRun): FlowTrack => {
  const laneY = LANE_Y[run.lane];
  const kfs: FlowKeyframe[] = [
    {
      t: run.emitAt - 160,
      x: BUCKET_X + 4,
      y: laneY,
      opacity: 0,
      scale: 0.5,
      state: "emitted",
    },
    {
      t: run.emitAt,
      x: BUCKET_X + 16,
      y: laneY,
      opacity: 1,
      scale: 1,
      ease: "out",
    },
    {
      t: run.arriveAt,
      x: slotX(run.enterDepth),
      y: laneY,
      ease: "linear",
      state: "queued",
    },
  ];
  // Shuffle toward the head as queue-mates are pulled; near-simultaneous
  // pulls are grouped into one multi-slot move so keyframes stay ordered.
  let depth = run.enterDepth;
  let i = 0;
  while (i < run.shuffles.length) {
    let j = i + 1;
    while (j < run.shuffles.length && run.shuffles[j] - run.shuffles[i] < 80)
      j++;
    const at = run.shuffles[i];
    const next = j < run.shuffles.length ? run.shuffles[j] : run.pullAt;
    const dur = Math.min(SHUFFLE_MS, (next - at) * 0.7);
    kfs.push({ t: at, x: slotX(depth), y: laneY, ease: "hold" });
    depth -= j - i;
    kfs.push({ t: at + dur, x: slotX(depth), y: laneY, ease: "out" });
    i = j;
  }
  kfs.push(
    { t: run.pullAt, x: slotX(0), y: laneY, ease: "hold" },
    {
      t: run.startAt,
      x: RUN_X,
      y: SLOT_Y[run.slot],
      ease: "inOut",
      state: "running",
    },
    {
      t: run.endAt,
      x: RUN_X,
      y: SLOT_Y[run.slot],
      ease: "hold",
      state: "done",
    },
    { t: run.endAt + DONE_DWELL, x: RUN_X, y: SLOT_Y[run.slot], ease: "hold" },
    {
      t: run.endAt + DONE_DWELL + EXIT_MS,
      x: TALLY_X,
      y: tallyY(run.tally),
      ease: "inOut",
      state: "tallied",
    },
    { t: FADE_AT, x: TALLY_X, y: tallyY(run.tally), ease: "hold" },
    {
      t: FADE_END,
      x: TALLY_X,
      y: tallyY(run.tally),
      opacity: 0,
      ease: "linear",
    },
  );
  return defineTrack(`s3p-obj-${run.index}`, kfs);
};

const OBJECT_TRACKS = RUNS.map((run) => ({
  lane: run.lane,
  track: objectTrack(run),
}));

// ─── list_objects trigger tracks (cron port → bucket, one per lane) ────────

const triggerTrack = (lane: number): FlowTrack => {
  const laneY = LANE_Y[lane];
  const arrive = TRIGGER_ARRIVE[lane];
  return defineTrack(`s3p-trig-${lane}`, [
    {
      t: CRON_AT,
      x: CRON_X + 6,
      y: MID_Y,
      opacity: 0,
      scale: 0.6,
      state: "trigger",
    },
    {
      t: CRON_AT + 180,
      x: CRON_X + 14,
      y: MID_Y + (laneY - MID_Y) * 0.12,
      opacity: 1,
      scale: 1,
      ease: "out",
    },
    // A midpoint keyframe approximates the fan curve the chrome draws.
    {
      t: arrive - 200,
      x: 40,
      y: MID_Y + (laneY - MID_Y) * 0.55,
      ease: "inOut",
    },
    { t: arrive, x: BUCKET_X - 14, y: laneY, ease: "out" },
    { t: arrive + 130, x: BUCKET_X - 14, y: laneY, ease: "hold" },
    {
      t: arrive + 380,
      x: BUCKET_X - 4,
      y: laneY,
      opacity: 0,
      scale: 0.5,
      ease: "in",
    },
  ]);
};

const TRIGGER_TRACKS = BUCKETS.map((_, lane) => triggerTrack(lane));

// ─── Cron pulse rings ──────────────────────────────────────────────────────

const pulseTrack = (n: number, delay: number): FlowTrack =>
  defineTrack(`s3p-pulse-${n}`, [
    { t: CRON_AT + delay, x: CRON_X, y: MID_Y, opacity: 0.9, scale: 0.4 },
    {
      t: CRON_AT + delay + 480,
      x: CRON_X,
      y: MID_Y,
      opacity: 0,
      scale: 2.2,
      ease: "out",
    },
  ]);

const PULSE_TRACKS = [pulseTrack(0, 0), pulseTrack(1, 140)];

// ─── Round-robin cursor (snaps to the queue head being pulled) ─────────────

const CURSOR_TRACK: FlowTrack = (() => {
  const kfs: FlowKeyframe[] = [
    {
      t: PULLS[0].pullAt - 420,
      x: HEAD_X,
      y: LANE_Y[PULLS[0].lane],
      opacity: 0,
    },
    {
      t: PULLS[0].pullAt - 240,
      x: HEAD_X,
      y: LANE_Y[PULLS[0].lane],
      opacity: 1,
      ease: "out",
    },
  ];
  let prev = PULLS[0].pullAt - 240;
  for (let i = 1; i < PULLS.length; i++) {
    const start = Math.max(prev + 40, PULLS[i].pullAt - 240);
    const arrive = Math.max(start + 60, PULLS[i].pullAt - 80);
    kfs.push({
      t: start,
      x: HEAD_X,
      y: LANE_Y[PULLS[i - 1].lane],
      ease: "hold",
    });
    kfs.push({ t: arrive, x: HEAD_X, y: LANE_Y[PULLS[i].lane], ease: "inOut" });
    prev = arrive;
  }
  const lastLane = LANE_Y[PULLS[PULLS.length - 1].lane];
  const lastPull = PULLS[PULLS.length - 1].pullAt;
  kfs.push(
    { t: lastPull + 500, x: HEAD_X, y: lastLane, ease: "hold" },
    { t: lastPull + 800, x: HEAD_X, y: lastLane, opacity: 0, ease: "linear" },
  );
  return defineTrack("s3p-cursor", kfs);
})();

// ─── Live readouts (per-frame, no React renders) ───────────────────────────

interface LoadFrame {
  backlog: number;
  inFlight: number;
  done: number;
  /** One per worker slot: how far through its object the worker is. */
  bars: { progress: number; opacity: number }[];
}

const sampleLoad = (t: number): LoadFrame => {
  let backlog = 0;
  let inFlight = 0;
  let done = 0;
  const bars = SLOT_Y.map(() => ({ progress: 0, opacity: 0 }));
  for (const run of RUNS) {
    if (t >= run.arriveAt && t < run.pullAt) backlog++;
    if (t >= run.pullAt && t < run.endAt) {
      inFlight++;
      const p = (t - run.startAt) / run.procMs;
      bars[run.slot] = { progress: Math.min(1, Math.max(0, p)), opacity: 1 };
    }
    if (t >= run.endAt) done++;
  }
  // After the tally fades the loop is back to its empty pre-tick state.
  if (t >= FADE_END) done = 0;
  return { backlog, inFlight, done, bars };
};

const backlogText = (f: LoadFrame) => `backlog · ${f.backlog}`;
const inFlightText = (f: LoadFrame) => `in flight · ${f.inFlight} / ${SLOTS}`;
const doneText = (f: LoadFrame) => `done · ${f.done}`;

const BACKLOG_LABEL_X = (slotX(Math.max(...MAX_DEPTH)) + HEAD_X) / 2;

/**
 * The three numbers that carry the argument — backlog under the queues,
 * in-flight under the worker, done under the tally — plus the per-slot
 * progress bars. All derived from the same schedule the tokens ride.
 */
const LiveLayer = () => {
  const { posterTime } = useFlow();
  const backlogRef = useRef<HTMLDivElement | null>(null);
  const inFlightRef = useRef<HTMLDivElement | null>(null);
  const doneRef = useRef<HTMLDivElement | null>(null);
  const barRefs = useRef<(HTMLDivElement | null)[]>([]);

  useFlowFrame((t) => {
    const f = sampleLoad(t);
    const backlog = backlogRef.current;
    if (backlog) {
      const text = backlogText(f);
      if (backlog.textContent !== text) backlog.textContent = text;
      const active = f.backlog > 0 ? "true" : "false";
      if (backlog.dataset.active !== active) backlog.dataset.active = active;
    }
    const inFlight = inFlightRef.current;
    if (inFlight) {
      const text = inFlightText(f);
      if (inFlight.textContent !== text) inFlight.textContent = text;
      const full = f.inFlight >= SLOTS ? "true" : "false";
      if (inFlight.dataset.full !== full) inFlight.dataset.full = full;
    }
    const done = doneRef.current;
    if (done) {
      const text = doneText(f);
      if (done.textContent !== text) done.textContent = text;
      const active = f.done > 0 ? "true" : "false";
      if (done.dataset.active !== active) done.dataset.active = active;
    }
    for (let s = 0; s < SLOTS; s++) {
      const bar = barRefs.current[s];
      if (!bar) continue;
      bar.style.width = `calc(var(--flow-u) * ${f.bars[s].progress * BAR_W})`;
      bar.style.opacity = String(f.bars[s].opacity);
    }
  });

  const initial = sampleLoad(posterTime);

  return (
    <>
      {SLOT_Y.map((y, s) => (
        <div
          key={y}
          ref={(el) => {
            barRefs.current[s] = el;
          }}
          className={styles.loadBar}
          style={{
            left: `calc(var(--flow-u) * ${BAR_X})`,
            top: `calc(var(--flow-u) * ${y - BAR_H / 2})`,
            width: `calc(var(--flow-u) * ${initial.bars[s].progress * BAR_W})`,
            opacity: initial.bars[s].opacity,
          }}
        />
      ))}
      <div
        ref={backlogRef}
        className={`${styles.stageLabel} ${styles.readout}`}
        data-active={initial.backlog > 0 ? "true" : "false"}
        style={{
          left: `calc(var(--flow-u) * ${BACKLOG_LABEL_X})`,
          top: `calc(var(--flow-u) * ${READOUT_Y})`,
        }}
      >
        {backlogText(initial)}
      </div>
      <div
        ref={inFlightRef}
        className={`${styles.stageLabel} ${styles.readout}`}
        data-full={initial.inFlight >= SLOTS ? "true" : "false"}
        style={{
          left: `calc(var(--flow-u) * ${WORKER_BOX.x + WORKER_BOX.w / 2})`,
          top: `calc(var(--flow-u) * ${READOUT_Y})`,
        }}
      >
        {inFlightText(initial)}
      </div>
      <div
        ref={doneRef}
        className={`${styles.stageLabel} ${styles.readout}`}
        data-active={initial.done > 0 ? "true" : "false"}
        style={{
          left: `calc(var(--flow-u) * ${TALLY_X})`,
          top: `calc(var(--flow-u) * ${READOUT_Y})`,
        }}
      >
        {doneText(initial)}
      </div>
    </>
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
    {/* Cron port + the fan to the three buckets */}
    <rect
      x={CRON_X - 2}
      y={MID_Y - 2}
      width={4}
      height={4}
      className={styles.chromeFill}
    />
    {LANE_Y.map((laneY) => (
      <path
        key={`fan-${laneY}`}
        d={`M ${CRON_X + 10} ${MID_Y} C 44 ${MID_Y}, 38 ${laneY}, ${BUCKET_X - 11} ${laneY}`}
        className={styles.chromeDash}
        {...fine}
      />
    ))}
    {/* Buckets: a tapered pail per lane */}
    {LANE_Y.map((laneY) => (
      <path
        key={`bucket-${laneY}`}
        d={`M ${BUCKET_X - 9} ${laneY - 7} L ${BUCKET_X + 9} ${laneY - 7} L ${BUCKET_X + 6} ${laneY + 7} L ${BUCKET_X - 6} ${laneY + 7} Z`}
        className={styles.chromeStroke}
        {...stroke}
      />
    ))}
    {/* Per-bucket lanes: outbound dash, queue rail with one tick per slot,
        and the pull stub into the worker */}
    {LANE_Y.map((laneY, lane) => (
      <g key={`lane-${lane}`}>
        <line
          x1={BUCKET_X + 12}
          y1={laneY}
          x2={slotX(MAX_DEPTH[lane]) - 10}
          y2={laneY}
          className={styles.chromeDash}
          {...fine}
        />
        <line
          x1={slotX(MAX_DEPTH[lane]) - 6}
          y1={laneY}
          x2={HEAD_X + 8}
          y2={laneY}
          className={styles.chromeRail}
          {...fine}
        />
        {Array.from({ length: MAX_DEPTH[lane] + 1 }, (_, d) => (
          <line
            key={d}
            x1={slotX(d)}
            y1={laneY + 4}
            x2={slotX(d)}
            y2={laneY + 7}
            className={styles.chromeTick}
            {...fine}
          />
        ))}
        <line
          x1={HEAD_X + 10}
          y1={laneY}
          x2={WORKER_BOX.x - 3}
          y2={laneY}
          className={styles.chromeDash}
          {...fine}
        />
      </g>
    ))}
    {/* Worker box with one ghost slot + bar track per unit of capacity */}
    <rect
      x={WORKER_BOX.x}
      y={WORKER_BOX.y}
      width={WORKER_BOX.w}
      height={WORKER_BOX.h}
      className={styles.chromeBox}
      {...stroke}
    />
    {SLOT_Y.map((y) => (
      <g key={y}>
        <rect
          x={RUN_X - 4.5}
          y={y - 4.5}
          width={9}
          height={9}
          className={styles.chromeGhost}
          {...fine}
        />
        <line
          x1={BAR_X}
          y1={y}
          x2={BAR_X + BAR_W}
          y2={y}
          className={styles.chromeGhost}
          strokeWidth={BAR_H}
          fill="none"
        />
      </g>
    ))}
    {/* Worker → processed column */}
    <line
      x1={WORKER_BOX.x + WORKER_BOX.w + 2}
      y1={MID_Y}
      x2={TALLY_X - 12}
      y2={MID_Y}
      className={styles.chromeDash}
      {...fine}
    />
    <path
      d={`M ${TALLY_X - 15} ${MID_Y - 3} L ${TALLY_X - 11} ${MID_Y} L ${TALLY_X - 15} ${MID_Y + 3}`}
      className={styles.chromeStroke}
      {...fine}
    />
    {/* Tally baseline */}
    <line
      x1={TALLY_X - 8}
      y1={TALLY_BASE + 5}
      x2={TALLY_X + 8}
      y2={TALLY_BASE + 5}
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
  lane,
  children,
}: {
  x: number;
  y: number;
  anchor?: "center" | "left";
  muted?: boolean;
  lane?: number;
  children: ReactNode;
}) => (
  <div
    className={[
      styles.stageLabel,
      anchor === "left" ? styles.anchorLeft : "",
      muted ? styles.labelMuted : "",
      lane !== undefined ? `${styles.laneLabel} ${styles[`b${lane}`]}` : "",
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
  "Animated diagram of the S3 processing pipeline. A cron tick fires at a port on the left and three hollow list_objects tasks fan out, one per S3 bucket. Each bucket then streams its objects into its own queue — bucket-1 holds the most and its queue grows deepest. A single worker with three slots pulls the queue heads round-robin, so its slots hold objects from different buckets at once, each with a progress bar. Finished objects settle into a processed column on the right whose interleaved colours show that no bucket monopolized the worker; when the backlog reaches zero the batch fades and the loop restarts.";

const CAPTION =
  "One cron tick fans out list_objects per bucket; a three-slot worker drains the queues round-robin, so bucket-1's backlog can't monopolize the pool.";

export const S3Pipeline = ({
  className,
  style,
  showCaption = true,
}: {
  className?: string;
  style?: CSSProperties;
  showCaption?: boolean;
}) => (
  <div
    className={[styles.wrap, className].filter(Boolean).join(" ")}
    style={style}
  >
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.flowRoot}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={6} y={86} anchor="left">
          cron
        </StageLabel>
        <StageLabel x={6} y={116} anchor="left" muted>
          list_objects
        </StageLabel>
        {BUCKETS.map((bucket, lane) => (
          <StageLabel
            key={bucket.name}
            x={BUCKET_X}
            y={LANE_Y[lane] - 17}
            lane={lane}
          >
            {bucket.name}
          </StageLabel>
        ))}
        <StageLabel x={WORKER_BOX.x + WORKER_BOX.w / 2} y={34}>
          worker
        </StageLabel>
        <LiveLayer />
        {PULSE_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.pulse} />
          </Flow.Token>
        ))}
        <Flow.Token track={CURSOR_TRACK}>
          <div className={styles.cursor} />
        </Flow.Token>
        {TRIGGER_TRACKS.map((track, lane) => (
          <Flow.Token
            key={track.id}
            track={track}
            className={styles[`b${lane}`]}
          >
            <div className={styles.trig} />
          </Flow.Token>
        ))}
        {OBJECT_TRACKS.map(({ lane, track }) => (
          <Flow.Token
            key={track.id}
            track={track}
            className={styles[`b${lane}`]}
          >
            <div className={styles.obj} />
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
