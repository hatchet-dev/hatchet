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
import styles from "./queuebackpressure.module.css";

/**
 * Flow animation for the moment the task queue enters the story: uploads
 * arrive faster than the worker can process them, and instead of the worker
 * falling over, the queue absorbs the burst.
 *
 * Beat 1 (calm) — two uploads trickle in and pass straight through the empty
 * tray onto the worker; backlog stays at 0.
 * Beat 2 (the flood) — ten uploads land in ~1.5s, far faster than the worker
 * drains them. The tray fills leftward, one slot per waiting task, and the
 * backlog readout climbs to 8.
 * Beat 3 (the payoff) — throughout, the worker holds exactly two tasks in
 * flight, one per capacity slot, each with its own progress bar. It never
 * takes a third upload just because ten arrived; it pulls the head of the
 * queue only when a slot frees, the tray shuffles right, and the backlog
 * drains back to 0 — the same state the loop starts in.
 *
 * The schedule is a tiny FIFO simulation run once at module scope (two worker
 * slots, fixed service time), so every pull time, queue depth and shuffle is
 * derived data rather than hand-typed magic — and identical on the server, on
 * the poster frame, and on every loop iteration. The only randomness is a
 * seeded ±30ms jitter on the burst arrivals so the stream does not read as a
 * metronome.
 */

// ─── Geometry (stage design units, 320 × 132 — wide inline composition) ────

const STAGE_W = 320;
const STAGE_H = 132;

/** Everything travels along one horizontal axis, left to right. */
const AXIS_Y = 76;

/** Upload source: a port on the left edge feeding the inbound lane. */
const SRC_X = 10;

/** Queue tray: slot 0 is the head (nearest the worker); it fills leftward. */
const HEAD_X = 174;
const SLOT_PITCH = 13;
const TRAY_TOP = 69;
const TRAY_BOTTOM = 83;
const slotX = (depth: number) => HEAD_X - depth * SLOT_PITCH;

/** Worker: one box, one row per capacity slot, each row a square + progress bar. */
const CONCURRENCY = 2;
const WORKER_BOX = { x: 228, y: 52, w: 50, h: 48 };
const RUN_X = 241;
const RUN_YS = [64, 88] as const;
const BAR_X = 250;
const BAR_W = 20;
const BAR_H = 2;

/** Completed tasks leave to the right and fade off the lane. */
const EXIT_X = 298;

// ─── Timing (ms) ───────────────────────────────────────────────────────────

const TRAVEL_IN = 650; // upload port → tail of the queue
const PULL_MS = 420; // head of the queue → worker slot
const PROC_MS = 1000; // one file processed on the worker
const SLOT_GAP = 100; // slot frees → next pull begins
const SHUFFLE_MS = 150; // queue shuffles one slot toward the head
const EXIT_MS = 460; // worker → off the right edge
const FADE_MS = 260;
const SPAWN_FADE = 140;

/** Beat 2: ten uploads in a row, each one landing before the last is served. */
const BURST_SIZE = 10;
const BURST_START = 3000;
const BURST_GAP = 150;

const jitter = createSeededRandom("queue-backpressure");

const SPAWNS: number[] = [
  200, // beat 1 — a calm pair, the pre-Bob steady state
  1500,
  ...Array.from({ length: BURST_SIZE }, (_, i) =>
    Math.round(BURST_START + i * BURST_GAP + (jitter() - 0.5) * 60)
  ),
];

// ─── Schedule (FIFO simulation, run once at module scope) ──────────────────

interface Task {
  index: number;
  /** Leaves the upload port. */
  spawnAt: number;
  /** Lands in the tray, at `enterDepth` slots from the head. */
  arriveAt: number;
  /** A worker slot frees and the head of the queue is dispatched. */
  pullAt: number;
  /** Processing starts — the task has reached its slot. */
  startAt: number;
  endAt: number;
  exitAt: number;
  slot: number;
  enterDepth: number;
  /** Pull times of the tasks ahead: each one shuffles this task one slot right. */
  shuffles: number[];
}

const TASKS: Task[] = (() => {
  const slotFree = Array.from({ length: CONCURRENCY }, () => 0);
  const out: Task[] = [];
  for (const [index, spawnAt] of SPAWNS.entries()) {
    const arriveAt = spawnAt + TRAVEL_IN;
    // FIFO: the head of the queue goes to whichever slot frees first.
    let slot = 0;
    for (let s = 1; s < CONCURRENCY; s++) if (slotFree[s] < slotFree[slot]) slot = s;
    const pullAt = Math.max(arriveAt, slotFree[slot]);
    const startAt = pullAt + PULL_MS;
    const endAt = startAt + PROC_MS;
    slotFree[slot] = endAt + SLOT_GAP;
    // Pull times are monotonic, so every earlier task still waiting when this
    // one lands is also dispatched before it — one shuffle each.
    const shuffles = out
      .filter((p) => p.pullAt > arriveAt && p.pullAt < pullAt)
      .map((p) => p.pullAt);
    out.push({
      index,
      spawnAt,
      arriveAt,
      pullAt,
      startAt,
      endAt,
      exitAt: endAt + EXIT_MS,
      slot,
      enterDepth: shuffles.length,
      shuffles,
    });
  }
  return out;
})();

/** How deep the burst actually gets — the tray is drawn exactly that deep. */
const MAX_DEPTH = Math.max(...TASKS.map((t) => t.enterDepth));
const TRAY_LEFT = slotX(MAX_DEPTH) - 7;
const TRAY_RIGHT = HEAD_X + 8;

const LAST_END = Math.max(...TASKS.map((t) => t.exitAt + FADE_MS));
const DURATION = Math.ceil((LAST_END + 400) / 200) * 200;

/**
 * Static frame: peak backlog. The tray is holding the burst, two more uploads
 * are still in the inbound lane, and the worker is at 2 of 2 — the whole
 * argument of the paragraph in a single frame.
 */
const POSTER_TIME = TASKS[TASKS.length - 3].arriveAt + 60;

// ─── Task tracks ───────────────────────────────────────────────────────────

/**
 * One token per upload: port → tail of the tray → shuffle toward the head as
 * tasks ahead are dispatched → a worker slot (accent while it runs) → out.
 */
const taskTrack = (task: Task): FlowTrack => {
  const kfs: FlowKeyframe[] = [
    { t: task.spawnAt, x: SRC_X, y: AXIS_Y, opacity: 0, state: "inflight" },
    { t: task.spawnAt + SPAWN_FADE, x: SRC_X + 8, y: AXIS_Y, opacity: 1, ease: "linear" },
  ];
  let depth = task.enterDepth;
  kfs.push({ t: task.arriveAt, x: slotX(depth), y: AXIS_Y, ease: "linear", state: "queued" });
  task.shuffles.forEach((at, i) => {
    // Never overrun the next event, however tightly two pulls land together.
    const next = task.shuffles[i + 1] ?? task.pullAt;
    const dur = Math.min(SHUFFLE_MS, (next - at) * 0.7);
    kfs.push({ t: at, x: slotX(depth), y: AXIS_Y, ease: "hold" });
    depth -= 1;
    kfs.push({ t: at + dur, x: slotX(depth), y: AXIS_Y, ease: "out" });
  });
  kfs.push(
    { t: task.pullAt, x: slotX(0), y: AXIS_Y, ease: "hold" },
    { t: task.startAt, x: RUN_X, y: RUN_YS[task.slot], ease: "inOut", state: "running" },
    { t: task.endAt, x: RUN_X, y: RUN_YS[task.slot], ease: "hold", state: "done" },
    { t: task.exitAt, x: EXIT_X, y: AXIS_Y, ease: "in" },
    { t: task.exitAt + FADE_MS, x: EXIT_X + 6, y: AXIS_Y, opacity: 0, ease: "linear" }
  );
  return defineTrack(`qbp-task-${task.index}`, kfs);
};

const TASK_TRACKS = TASKS.map(taskTrack);

// ─── Live readouts (per-frame, no React renders) ───────────────────────────

interface LoadFrame {
  backlog: number;
  inFlight: number;
  /** One per capacity slot: how far through its file the worker is. */
  bars: { progress: number; opacity: number }[];
}

const sampleLoad = (t: number): LoadFrame => {
  let backlog = 0;
  let inFlight = 0;
  const bars = RUN_YS.map(() => ({ progress: 0, opacity: 0 }));
  for (const task of TASKS) {
    if (t >= task.arriveAt && t < task.pullAt) backlog++;
    if (t >= task.pullAt && t < task.endAt) {
      inFlight++;
      const p = (t - task.startAt) / PROC_MS;
      bars[task.slot] = { progress: Math.min(1, Math.max(0, p)), opacity: 1 };
    }
  }
  return { backlog, inFlight, bars };
};

const backlogText = (f: LoadFrame) => `backlog · ${f.backlog}`;
const inFlightText = (f: LoadFrame) => `in flight · ${f.inFlight} / ${CONCURRENCY}`;

/**
 * The two numbers that carry the argument — backlog under the tray, in-flight
 * under the worker — plus the per-slot progress bars. All of it is derived
 * from the same schedule the tokens ride, written imperatively each frame.
 */
const LiveLayer = () => {
  const { posterTime } = useFlow();
  const backlogRef = useRef<HTMLDivElement | null>(null);
  const inFlightRef = useRef<HTMLDivElement | null>(null);
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
      const full = f.inFlight >= CONCURRENCY ? "true" : "false";
      if (inFlight.dataset.full !== full) inFlight.dataset.full = full;
    }
    for (let s = 0; s < CONCURRENCY; s++) {
      const bar = barRefs.current[s];
      if (!bar) continue;
      bar.style.width = `calc(var(--flow-u) * ${f.bars[s].progress * BAR_W})`;
      bar.style.opacity = String(f.bars[s].opacity);
    }
  });

  const initial = sampleLoad(posterTime);

  return (
    <>
      {RUN_YS.map((y, s) => (
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
          left: `calc(var(--flow-u) * ${(TRAY_LEFT + TRAY_RIGHT) / 2})`,
          top: `calc(var(--flow-u) * 110)`,
        }}
      >
        {backlogText(initial)}
      </div>
      <div
        ref={inFlightRef}
        className={`${styles.stageLabel} ${styles.readout}`}
        data-full={initial.inFlight >= CONCURRENCY ? "true" : "false"}
        style={{
          left: `calc(var(--flow-u) * ${WORKER_BOX.x + WORKER_BOX.w / 2})`,
          top: `calc(var(--flow-u) * 110)`,
        }}
      >
        {inFlightText(initial)}
      </div>
    </>
  );
};

// ─── Static chrome ─────────────────────────────────────────────────────────

const stroke = { fill: "none", strokeWidth: 1.5, vectorEffect: "non-scaling-stroke" } as const;
const fine = { fill: "none", strokeWidth: 1, vectorEffect: "non-scaling-stroke" } as const;

const Chrome = () => (
  <svg viewBox={`0 0 ${STAGE_W} ${STAGE_H}`} aria-hidden="true">
    {/* Upload port + inbound lane */}
    <rect x={6} y={AXIS_Y - 2} width={4} height={4} className={styles.chromeFill} />
    <line
      x1={14}
      y1={AXIS_Y}
      x2={TRAY_LEFT - 4}
      y2={AXIS_Y}
      className={styles.chromeDash}
      {...fine}
    />
    {/* Queue tray: brackets at both ends, one capacity tick per slot */}
    <path
      d={`M ${TRAY_LEFT + 6} ${TRAY_TOP} L ${TRAY_LEFT} ${TRAY_TOP} L ${TRAY_LEFT} ${TRAY_BOTTOM} L ${TRAY_LEFT + 6} ${TRAY_BOTTOM}`}
      className={styles.chromeStroke}
      {...stroke}
    />
    <path
      d={`M ${TRAY_RIGHT - 6} ${TRAY_TOP} L ${TRAY_RIGHT} ${TRAY_TOP} L ${TRAY_RIGHT} ${TRAY_BOTTOM} L ${TRAY_RIGHT - 6} ${TRAY_BOTTOM}`}
      className={styles.chromeStroke}
      {...stroke}
    />
    {Array.from({ length: MAX_DEPTH + 1 }, (_, d) => (
      <line
        key={d}
        x1={slotX(d)}
        y1={TRAY_BOTTOM - 3}
        x2={slotX(d)}
        y2={TRAY_BOTTOM}
        className={styles.chromeTick}
        {...fine}
      />
    ))}
    {/* Tray → worker: the dispatch lane */}
    <line
      x1={TRAY_RIGHT + 2}
      y1={AXIS_Y}
      x2={WORKER_BOX.x - 2}
      y2={AXIS_Y}
      className={styles.chromeDash}
      {...fine}
    />
    {/* Worker box with one ghost slot + bar track per unit of capacity */}
    <rect
      x={WORKER_BOX.x}
      y={WORKER_BOX.y}
      width={WORKER_BOX.w}
      height={WORKER_BOX.h}
      className={styles.chromeBox}
      {...stroke}
    />
    {RUN_YS.map((y) => (
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
    {/* Worker → done */}
    <path
      d={`M ${WORKER_BOX.x + WORKER_BOX.w + 2} ${AXIS_Y} H ${EXIT_X + 8}`}
      className={styles.chromeDash}
      {...fine}
    />
    <path
      d={`M ${EXIT_X + 4} ${AXIS_Y - 3} L ${EXIT_X + 8} ${AXIS_Y} L ${EXIT_X + 4} ${AXIS_Y + 3}`}
      className={styles.chromeStroke}
      {...fine}
    />
  </svg>
);

const StageLabel = ({
  x,
  y,
  anchor = "center",
  children,
}: {
  x: number;
  y: number;
  anchor?: "center" | "left";
  children: ReactNode;
}) => (
  <div
    className={`${styles.stageLabel} ${anchor === "left" ? styles.anchorLeft : ""}`}
    style={{ left: `calc(var(--flow-u) * ${x})`, top: `calc(var(--flow-u) * ${y})` }}
  >
    {children}
  </div>
);

// ─── Export ────────────────────────────────────────────────────────────────

const ARIA_LABEL =
  "Animated diagram of a task queue absorbing a burst of uploads. Upload tasks arrive faster than the worker can process them, so they pile up in the queue as backlog instead of hitting the worker all at once. The worker pulls the head of the queue only when one of its two slots frees, so it never runs more than two files at a time, and the backlog drains at the worker's own pace.";

const CAPTION =
  "The queue absorbs the burst: uploads pile up as backlog while the worker keeps to the two tasks it can run at a time.";

export const QueueBackpressure = ({
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
        <StageLabel x={6} y={20} anchor="left">
          uploads
        </StageLabel>
        <StageLabel x={(TRAY_LEFT + TRAY_RIGHT) / 2} y={20}>
          task queue
        </StageLabel>
        <StageLabel x={WORKER_BOX.x + WORKER_BOX.w / 2} y={20}>
          worker
        </StageLabel>
        <StageLabel x={EXIT_X} y={20}>
          done
        </StageLabel>
        <LiveLayer />
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
