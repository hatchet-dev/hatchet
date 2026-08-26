"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  createSeededRandom,
  defineTrack,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import styles from "./prefetchblocking.module.css";

/**
 * Flow animation of Celery's default worker prefetch count
 * (`worker_prefetch_multiplier` = 4), for the "Worker Prefetch Count" section
 * of the problems-with-celery post.
 *
 * Worker 1 has a single concurrency slot, so its prefetch count is 4: it may
 * hold four unacknowledged messages at once. The broker queue holds exactly
 * four tasks — one long, three short — so worker 1 greedily drains the whole
 * queue into its own local buffer in one burst, before it has executed
 * anything. It then executes the long task in its one slot; the three short
 * tasks it already claimed sit reserved behind it, unacked and therefore not
 * redeliverable to anyone else. Worker 2 is online and free the entire time,
 * with an empty buffer, because the queue was already drained. When the long
 * task finally finishes, worker 1 works through the three short tasks
 * serially — work that the idle worker could have finished long before.
 *
 * The fourth buffer slot empties as soon as the long task starts executing
 * (early ack), and stays empty: the worker is willing to fetch more, there is
 * simply nothing left in the queue. That is the failure the section describes
 * — prefetching converts a queueing problem into wasted capacity.
 *
 * Timings are hand-scheduled at module scope with one seeded PRNG for jitter,
 * so the loop is fully deterministic and the composition at t = DURATION
 * matches t = 0 (the queue refills off the left edge to close the loop).
 */

// ─── Geometry (stage design units, 320 × 208 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 208;

/** Broker queue: a bracketed column of four slots, head at the top. */
const QUEUE_X = 34;
const QUEUE_YS = [84, 104, 124, 144] as const;
const QUEUE_TOP = 70;
const QUEUE_BOTTOM = 158;

/** Where the delivery lines leave the queue and fan out to both workers. */
const FEED_X = 60;

/** Worker boxes: identical capacity, stacked. Worker 1 on top. */
const WORKER_X0 = 76;
const WORKER_X1 = 306;
const W1 = { top: 40, bottom: 100, row: 70 } as const;
const W2 = { top: 128, bottom: 188, row: 158 } as const;

/** Prefetch buffer: four reserved positions inside each worker box. */
const BUF_XS = [102, 126, 150, 174] as const;
const DIVIDER_X = 198;

/** The single concurrency slot — one execution cell per worker (46 × 32 in CSS). */
const EXEC_X = 252;

/** The link that never carries anything: buffer 1 → buffer 2. Sits right of
    the "worker 2" label so the cross never crowds it. */
const BLOCK_X = 162;
const BLOCK_Y = 114;

/** Centre of the four buffer positions — where the buffer labels sit. */
const BUF_MID = (BUF_XS[0] + BUF_XS[3]) / 2;

/** Baselines for the in-box labels under each worker's buffer and slot. */
const LABEL_Y1 = W1.bottom - 12;
const LABEL_Y2 = W2.bottom - 12;

// ─── Timing (ms) ───────────────────────────────────────────────────────────

const rng = createSeededRandom("prefetch-blocking");

const BURST_START = 700; // first delivery leaves the queue
const BURST_GAP = 215; // deliveries are a burst, not a trickle
const TRAVEL_IN = 700; // queue → prefetch buffer
const SHIFT_MS = 380; // buffer advances one position
const PICKUP_MS = 320; // buffer head → execution cell
const HOLD_DONE = 120; // finished task lingers before it clears
const FADE_MS = 380;
const NEXT_GAP = 120; // cell clears → worker pulls the next reserved task
const LONG_MS = 5200; // the one long-running task
const FIRST_PICKUP = 2500; // all four are buffered well before this

interface Task {
  /** Queue slot / buffer arrival position (they are the same index). */
  index: number;
  long: boolean;
  depart: number; // leaves its queue slot
  land: number; // lands in the prefetch buffer
  pickup: number; // leaves the buffer head for the execution cell
  start: number; // begins executing
  done: number; // finishes
  gone: number; // fully faded from the cell
}

/**
 * One long task followed by three short ones, all four delivered up-front.
 * Each pickup waits for the previous task to clear the single slot, so the
 * shorts run strictly serially on the busy worker.
 */
const TASKS: Task[] = [];
{
  let pickup = FIRST_PICKUP;
  for (let index = 0; index < 4; index++) {
    const long = index === 0;
    const depart = BURST_START + index * BURST_GAP + Math.round(rng() * 60);
    const start = pickup + PICKUP_MS;
    const done = start + (long ? LONG_MS : 620 + Math.round(rng() * 170));
    const gone = done + HOLD_DONE + FADE_MS;
    TASKS.push({
      index,
      long,
      depart,
      land: depart + TRAVEL_IN,
      pickup,
      start,
      done,
      gone,
    });
    pickup = gone + NEXT_GAP;
  }
}

const LAST = TASKS[TASKS.length - 1];

/** Worker 2 is provably starved from the moment worker 1 is committed. */
const STARVE_FROM = TASKS[0].start + 200;
const STARVE_TO = LAST.done;

/** The queue refills off the left edge so the loop wraps onto its own start. */
const REFILL_AT = LAST.gone + 200;
const REFILL_GAP = 150;
const DURATION = REFILL_AT + (TASKS.length - 1) * REFILL_GAP + TRAVEL_IN + 520;

// Resulting storyboard (ms, DURATION ≈ 15400):
//
//    0        four tasks queued in the broker — 1 long, 3 short
//    0.7–2.1  worker 1 drains all four into its prefetch buffer
//    2.5–2.8  the long task takes the single slot; the buffer advances
//    2.8–8.0  long task runs; three short tasks sit reserved; worker 2 idle
//    8.6–13.0 the shorts finally run, one at a time, still on worker 1
//   13.7–15.4 the queue refills off-stage left and the loop wraps

/**
 * Static frame: the long task mid-flight in worker 1's only slot, the three
 * short tasks visibly stuck in its buffer, the queue empty, and worker 2
 * sitting idle with the "can't be reassigned" link struck through. This is the
 * whole argument in one picture, so it is what crawlers and reduced-motion
 * readers get.
 */
const POSTER_TIME = 5200;

// ─── Track builders ────────────────────────────────────────────────────────

/**
 * One token per task: queued in the broker → prefetched into worker 1's local
 * buffer → shuffled forward as the buffer head is picked up → executed in the
 * single slot → cleared → re-queued off-stage left to close the loop.
 */
const taskTrack = (task: Task): FlowTrack => {
  const queueY = QUEUE_YS[task.index];
  const kfs: FlowKeyframe[] = [
    { t: 0, x: QUEUE_X, y: queueY, opacity: 1, scale: 1, state: "queued" },
    { t: task.depart, x: QUEUE_X, y: queueY, ease: "hold" },
    { t: task.depart + 260, x: FEED_X, y: W1.row, ease: "in" },
    {
      t: task.land,
      x: BUF_XS[task.index],
      y: W1.row,
      ease: "out",
      state: "reserved",
    },
  ];
  // Every earlier pickup advances this task one position toward the head.
  for (let j = 0; j < task.index; j++) {
    const from = task.index - j;
    kfs.push(
      { t: TASKS[j].pickup, x: BUF_XS[from], y: W1.row, ease: "hold" },
      {
        t: TASKS[j].pickup + SHIFT_MS,
        x: BUF_XS[from - 1],
        y: W1.row,
        ease: "inOut",
      },
    );
  }
  kfs.push(
    { t: task.pickup, x: BUF_XS[0], y: W1.row, ease: "hold" },
    { t: task.start, x: EXEC_X, y: W1.row, ease: "inOut", state: "executing" },
    { t: task.done, x: EXEC_X, y: W1.row, ease: "hold", state: "done" },
    { t: task.done + HOLD_DONE, x: EXEC_X, y: W1.row, ease: "hold" },
    { t: task.gone, x: EXEC_X, y: W1.row, opacity: 0, scale: 1.3, ease: "out" },
  );
  // Re-queue invisibly off the left edge, then slide back into the slot the
  // loop started from — t = DURATION is byte-for-byte the t = 0 composition.
  const refill = REFILL_AT + task.index * REFILL_GAP;
  kfs.push(
    {
      t: refill,
      x: -16,
      y: queueY,
      opacity: 0,
      scale: 1,
      ease: "hold",
      state: "queued",
    },
    { t: refill + TRAVEL_IN, x: QUEUE_X, y: queueY, opacity: 1, ease: "out" },
    { t: DURATION, x: QUEUE_X, y: queueY, ease: "hold" },
  );
  return defineTrack(`prefetch-task-${task.index}`, kfs);
};

const TASK_TRACKS = TASKS.map(taskTrack);

/** Worker 1's slot: busy for exactly one task at a time, all loop long. */
const EXEC_1 = defineTrack("prefetch-exec-1", [
  { t: 0, x: EXEC_X, y: W1.row, state: "idle" },
  ...TASKS.flatMap((task): FlowKeyframe[] => [
    { t: task.start, x: EXEC_X, y: W1.row, ease: "hold", state: "busy" },
    { t: task.done, x: EXEC_X, y: W1.row, ease: "hold", state: "idle" },
  ]),
  { t: DURATION, x: EXEC_X, y: W1.row, ease: "hold" },
]);

/** Worker 2's slot: never busy — it flips to "starved" and stays there. */
const EXEC_2 = defineTrack("prefetch-exec-2", [
  { t: 0, x: EXEC_X, y: W2.row, state: "idle" },
  { t: STARVE_FROM, x: EXEC_X, y: W2.row, ease: "hold", state: "starved" },
  { t: STARVE_TO, x: EXEC_X, y: W2.row, ease: "hold", state: "idle" },
  { t: DURATION, x: EXEC_X, y: W2.row, ease: "hold" },
]);

/**
 * The "idle" tag under worker 2's slot, which goes magenta while it starves. Tokens
 * are centered on their y, static labels hang from theirs, hence the nudge —
 * this sits on the same visual line as the "empty" label beside it.
 */
const IDLE_TAG_Y = LABEL_Y2 + 2.5;
const IDLE_TAG = defineTrack("prefetch-idle-tag", [
  { t: 0, x: EXEC_X, y: IDLE_TAG_Y, state: "idle" },
  { t: STARVE_FROM, x: EXEC_X, y: IDLE_TAG_Y, ease: "hold", state: "starved" },
  { t: STARVE_TO, x: EXEC_X, y: IDLE_TAG_Y, ease: "hold", state: "idle" },
  { t: DURATION, x: EXEC_X, y: IDLE_TAG_Y, ease: "hold" },
]);

/** Callout on the dead link between the two buffers — visible only while it bites. */
const callout = (id: string, x: number, y: number): FlowTrack =>
  defineTrack(id, [
    { t: 0, x, y, opacity: 0 },
    { t: STARVE_FROM, x, y, opacity: 0, ease: "hold" },
    { t: STARVE_FROM + 260, x, y, opacity: 1, ease: "linear" },
    { t: STARVE_TO, x, y, ease: "hold" },
    { t: STARVE_TO + 260, x, y, opacity: 0, ease: "linear" },
    { t: DURATION, x, y, ease: "hold" },
  ]);

const BLOCK_MARK = callout("prefetch-block-mark", BLOCK_X, BLOCK_Y);
const BLOCK_TAG = callout("prefetch-block-tag", 234, BLOCK_Y - 1);

/**
 * The tag under the queue swaps with the queue's contents: it describes the
 * workload while the four tasks are sitting there, and reads "drained" for the
 * long stretch when worker 1 has taken all of them — which is exactly why
 * worker 2 has nothing to pull.
 */
const QUEUE_TAG_Y = 166;
const QUEUE_EMPTY_AT = LAST.depart + 300;
const QUEUE_REFILLED_AT = DURATION - 760;

const WORKLOAD_TAG = defineTrack("prefetch-workload-tag", [
  { t: 0, x: QUEUE_X, y: QUEUE_TAG_Y, opacity: 1 },
  { t: QUEUE_EMPTY_AT, x: QUEUE_X, y: QUEUE_TAG_Y, ease: "hold" },
  {
    t: QUEUE_EMPTY_AT + 220,
    x: QUEUE_X,
    y: QUEUE_TAG_Y,
    opacity: 0,
    ease: "linear",
  },
  { t: QUEUE_REFILLED_AT, x: QUEUE_X, y: QUEUE_TAG_Y, ease: "hold" },
  {
    t: QUEUE_REFILLED_AT + 300,
    x: QUEUE_X,
    y: QUEUE_TAG_Y,
    opacity: 1,
    ease: "linear",
  },
  { t: DURATION, x: QUEUE_X, y: QUEUE_TAG_Y, ease: "hold" },
]);

const DRAINED_TAG = defineTrack("prefetch-drained-tag", [
  { t: 0, x: QUEUE_X, y: QUEUE_TAG_Y, opacity: 0 },
  { t: QUEUE_EMPTY_AT + 220, x: QUEUE_X, y: QUEUE_TAG_Y, ease: "hold" },
  {
    t: QUEUE_EMPTY_AT + 520,
    x: QUEUE_X,
    y: QUEUE_TAG_Y,
    opacity: 1,
    ease: "linear",
  },
  { t: QUEUE_REFILLED_AT - 300, x: QUEUE_X, y: QUEUE_TAG_Y, ease: "hold" },
  {
    t: QUEUE_REFILLED_AT,
    x: QUEUE_X,
    y: QUEUE_TAG_Y,
    opacity: 0,
    ease: "linear",
  },
  { t: DURATION, x: QUEUE_X, y: QUEUE_TAG_Y, ease: "hold" },
]);

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

/** The four prefetch positions drawn as empty capacity, one worker's worth. */
const BufferSlots = ({ row }: { row: number }) => (
  <>
    {BUF_XS.map((x) => (
      <rect
        key={x}
        x={x - 4}
        y={row - 4}
        width={8}
        height={8}
        className={styles.chromeSlot}
        {...fine}
      />
    ))}
  </>
);

const WorkerBox = ({
  top,
  bottom,
  row,
}: {
  top: number;
  bottom: number;
  row: number;
}) => (
  <>
    <rect
      x={WORKER_X0}
      y={top}
      width={WORKER_X1 - WORKER_X0}
      height={bottom - top}
      className={styles.chromeBox}
      {...stroke}
    />
    <BufferSlots row={row} />
    {/* Buffer | execution slot divider */}
    <line
      x1={DIVIDER_X}
      y1={top + 8}
      x2={DIVIDER_X}
      y2={bottom - 8}
      className={styles.chromeDash}
      {...fine}
    />
  </>
);

const Chrome = () => (
  <svg
    viewBox={`0 0 ${STAGE_W} ${STAGE_H}`}
    aria-hidden="true"
    className={styles.chrome}
  >
    {/* Broker queue: brackets around four slots */}
    <path
      d={`M ${QUEUE_X - 15} ${QUEUE_TOP + 5} L ${QUEUE_X - 15} ${QUEUE_TOP} L ${QUEUE_X + 15} ${QUEUE_TOP} L ${QUEUE_X + 15} ${QUEUE_TOP + 5}`}
      className={styles.chromeStroke}
      {...stroke}
    />
    <path
      d={`M ${QUEUE_X - 15} ${QUEUE_BOTTOM - 5} L ${QUEUE_X - 15} ${QUEUE_BOTTOM} L ${QUEUE_X + 15} ${QUEUE_BOTTOM} L ${QUEUE_X + 15} ${QUEUE_BOTTOM - 5}`}
      className={styles.chromeStroke}
      {...stroke}
    />
    {/* Delivery lines: one queue, two consumers — worker 2 is connected too */}
    <path
      d={`M ${QUEUE_X + 16} 114 H ${FEED_X} V ${W1.row} H ${WORKER_X0 - 4}`}
      className={styles.chromeDash}
      {...fine}
    />
    <path
      d={`M ${FEED_X} 114 V ${W2.row} H ${WORKER_X0 - 4}`}
      className={styles.chromeDash}
      {...fine}
    />
    <WorkerBox {...W1} />
    <WorkerBox {...W2} />
    {/* The link that prefetch severs: reserved work can't cross to worker 2 */}
    <line
      x1={BLOCK_X}
      y1={W1.bottom}
      x2={BLOCK_X}
      y2={W2.top}
      className={styles.chromeDeadLink}
      {...fine}
    />
  </svg>
);

/**
 * Labels are set in --micro, which does not scale with the stage, so every
 * label is kept short enough that a row of them still fits side by side on a
 * phone-width stage.
 */
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
  "Animated diagram of Celery's default worker prefetch count. Worker 1 has one concurrency slot and a prefetch multiplier of 4, so it pulls all four queued tasks — one long, three short — into its own local buffer at once. It runs the long task in its single slot while the three short tasks it already claimed sit reserved behind it, unacknowledged and so unable to be reassigned. Worker 2 stays idle with an empty buffer for the entire run, because the queue was already drained.";

export const PrefetchBlocking = ({ style }: { style?: CSSProperties }) => (
  <div className={styles.wrap} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.flowRoot}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={QUEUE_X} y={20}>
          broker
          <br />
          queue
        </StageLabel>
        <StageLabel x={WORKER_X0} y={26} anchor="left">
          worker 1 · prefetch 4
        </StageLabel>
        <StageLabel x={WORKER_X0} y={110} anchor="left">
          worker 2
        </StageLabel>
        <StageLabel x={BUF_MID} y={LABEL_Y1} muted>
          reserved
        </StageLabel>
        <StageLabel x={EXEC_X} y={LABEL_Y1} muted>
          executing
        </StageLabel>
        <StageLabel x={BUF_MID} y={LABEL_Y2} muted>
          empty
        </StageLabel>
        <Flow.Token track={WORKLOAD_TAG}>
          <div className={styles.queueTag}>1 long · 3 short</div>
        </Flow.Token>
        <Flow.Token track={DRAINED_TAG}>
          <div className={styles.queueTag}>drained</div>
        </Flow.Token>
        <Flow.Token track={EXEC_1}>
          <div className={styles.execCell} />
        </Flow.Token>
        <Flow.Token track={EXEC_2}>
          <div className={styles.execCell} />
        </Flow.Token>
        <Flow.Token track={IDLE_TAG}>
          <div className={styles.idleTag}>idle</div>
        </Flow.Token>
        <Flow.Token track={BLOCK_MARK}>
          <div className={styles.blockMark} />
        </Flow.Token>
        <Flow.Token track={BLOCK_TAG}>
          <div className={styles.blockTag}>can&rsquo;t be reassigned</div>
        </Flow.Token>
        {TASK_TRACKS.map((track, i) => (
          <Flow.Token key={track.id} track={track}>
            <div
              className={
                TASKS[i].long
                  ? `${styles.task} ${styles.taskLong}`
                  : styles.task
              }
            />
          </Flow.Token>
        ))}
      </Flow.Stage>
    </Flow.Root>
  </div>
);
