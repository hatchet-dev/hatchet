"use client";

import { useRef, type CSSProperties, type ReactNode } from "react";
import { Flow, defineTrack, useFlow, useFlowFrame } from "@/components/flow";
import styles from "./childcycle.module.css";

/**
 * Flow animation of an agent reasoning loop
 * (docs.hatchet.run/v1/child-spawning): a durable parent spawns a child run
 * of itself, the child runs, and the result comes back to a termination
 * check. Twice the verdict is "no — run again" and the token rides the
 * loop-back rail into the parent to spawn the next iteration; on the third
 * pass the condition is met and the token exits right into the complete box.
 * An iteration readout under the loop counts 1 → 3.
 *
 * One token cycles the whole loop — every iteration is the same shape, which
 * is the point. All timings are hand-scheduled at module scope, so the loop
 * is fully deterministic and t = DURATION matches the empty stage at t = 0.
 */

// ─── Geometry (stage design units, 320 × 200 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 200;

/** The durable parent on the left. */
const PARENT_X = 56;
const PARENT_Y = 84;

/** Child run slot in the middle of the forward rail. */
const CHILD_X = 156;
const CHILD_Y = 84;

/** Termination check diamond. */
const CHECK_X = 238;
const CHECK_Y = 84;

/** Complete box on the right edge. */
const DONE_X = 294;
const DONE_Y = 84;

/** Loop-back rail: check → down → left → up into the parent. */
const LOOP_Y = 140;

const READOUT_X = 147;
const READOUT_Y = 166;
const LOOP_VERDICT = { x: 147, y: 148 };
const DONE_VERDICT = { x: 266, y: 68 };

// ─── Timing (ms) — three spawn → run → check cycles, exit on the third ─────

const SPAWN_MS = 450; // parent → child slot
const RUN_MS = 1150; // one child invocation
const TO_CHECK_MS = 350; // child slot → check
const CHECK_DWELL = 550; // the verdict beat
const LOOP_DOWN = 250;
const LOOP_ACROSS = 600;
const LOOP_UP = 250;

/** Loop times at which each iteration leaves the parent (200ms respawn dwell). */
const ITER_STARTS = [500, 4300, 8100] as const;
const ITERATIONS = ITER_STARTS.length;

const DONE_AT = 10950; // final verdict + 350ms exit: the token lands in complete
const FADE_START = 12100;
const FADE_END = 12400;
const DURATION = 12800;

/** Static frame: iteration 2 mid-run — the loop is the story, mid-loop. */
const POSTER_TIME = 5300;

// ─── Tracks ────────────────────────────────────────────────────────────────

/** Forward leg + check for iteration starting at `t0`. */
const cycleStart = (t0: number) =>
  [
    {
      t: t0 + SPAWN_MS,
      x: CHILD_X,
      y: CHILD_Y,
      ease: "inOut",
      state: "running",
    },
    { t: t0 + SPAWN_MS + RUN_MS, x: CHILD_X, y: CHILD_Y, ease: "hold" },
    {
      t: t0 + SPAWN_MS + RUN_MS + TO_CHECK_MS,
      x: CHECK_X,
      y: CHECK_Y,
      ease: "inOut",
      state: "checking",
    },
  ] as const;

/** Loop-back leg: check verdict says run again. */
const loopBack = (checkEnd: number) =>
  [
    { t: checkEnd, x: CHECK_X, y: CHECK_Y, ease: "hold", state: "looping" },
    { t: checkEnd + LOOP_DOWN, x: CHECK_X, y: LOOP_Y, ease: "in" },
    {
      t: checkEnd + LOOP_DOWN + LOOP_ACROSS,
      x: PARENT_X,
      y: LOOP_Y,
      ease: "linear",
    },
    {
      t: checkEnd + LOOP_DOWN + LOOP_ACROSS + LOOP_UP,
      x: PARENT_X,
      y: PARENT_Y,
      ease: "out",
    },
  ] as const;

const checkEnd = (t0: number) =>
  t0 + SPAWN_MS + RUN_MS + TO_CHECK_MS + CHECK_DWELL;

/** The one token that rides all three iterations. */
const RUN_TRACK = defineTrack("cc-run", [
  // Iteration 1: fade in at the parent, run, check → loop.
  {
    t: ITER_STARTS[0] - 200,
    x: PARENT_X,
    y: PARENT_Y,
    opacity: 0,
    scale: 0.7,
    state: "spawned",
  },
  {
    t: ITER_STARTS[0],
    x: PARENT_X,
    y: PARENT_Y,
    opacity: 1,
    scale: 1,
    ease: "out",
  },
  ...cycleStart(ITER_STARTS[0]),
  ...loopBack(checkEnd(ITER_STARTS[0])),
  // Iteration 2: dwell at the parent, out again → loop.
  {
    t: ITER_STARTS[1],
    x: PARENT_X,
    y: PARENT_Y,
    ease: "hold",
    state: "spawned",
  },
  ...cycleStart(ITER_STARTS[1]),
  ...loopBack(checkEnd(ITER_STARTS[1])),
  // Iteration 3: condition met — exit right to complete.
  {
    t: ITER_STARTS[2],
    x: PARENT_X,
    y: PARENT_Y,
    ease: "hold",
    state: "spawned",
  },
  ...cycleStart(ITER_STARTS[2]),
  {
    t: checkEnd(ITER_STARTS[2]),
    x: CHECK_X,
    y: CHECK_Y,
    ease: "hold",
    state: "complete",
  },
  { t: DONE_AT, x: DONE_X, y: DONE_Y, ease: "out" },
  { t: FADE_START, x: DONE_X, y: DONE_Y, ease: "hold" },
  { t: FADE_END, x: DONE_X, y: DONE_Y, opacity: 0, scale: 1.3, ease: "out" },
]);

/** The parent box: waiting on the loop from first spawn to final verdict. */
const PARENT_TRACK = defineTrack("cc-parent", [
  { t: 0, x: PARENT_X, y: PARENT_Y, state: "idle" },
  {
    t: ITER_STARTS[0],
    x: PARENT_X,
    y: PARENT_Y,
    ease: "hold",
    state: "active",
  },
  {
    t: checkEnd(ITER_STARTS[2]),
    x: PARENT_X,
    y: PARENT_Y,
    ease: "hold",
    state: "idle",
  },
  { t: DURATION, x: PARENT_X, y: PARENT_Y, ease: "hold" },
]);

/** The complete box: lights up when the final result lands. */
const DONE_TRACK = defineTrack("cc-done", [
  { t: 0, x: DONE_X, y: DONE_Y, state: "idle" },
  { t: DONE_AT, x: DONE_X, y: DONE_Y, ease: "hold", state: "reached" },
  { t: FADE_END, x: DONE_X, y: DONE_Y, ease: "hold", state: "idle" },
  { t: DURATION, x: DONE_X, y: DONE_Y, ease: "hold" },
]);

/** "no · run again" flashes under the loop rail at the first two verdicts. */
const LOOP_VERDICT_TRACK = defineTrack("cc-verdict-loop", [
  {
    t: checkEnd(ITER_STARTS[0]) - 500,
    x: LOOP_VERDICT.x,
    y: LOOP_VERDICT.y,
    opacity: 0,
  },
  {
    t: checkEnd(ITER_STARTS[0]) - 350,
    x: LOOP_VERDICT.x,
    y: LOOP_VERDICT.y,
    opacity: 1,
    ease: "linear",
  },
  {
    t: checkEnd(ITER_STARTS[0]) + 400,
    x: LOOP_VERDICT.x,
    y: LOOP_VERDICT.y,
    ease: "hold",
  },
  {
    t: checkEnd(ITER_STARTS[0]) + 650,
    x: LOOP_VERDICT.x,
    y: LOOP_VERDICT.y,
    opacity: 0,
    ease: "linear",
  },
  {
    t: checkEnd(ITER_STARTS[1]) - 500,
    x: LOOP_VERDICT.x,
    y: LOOP_VERDICT.y,
    opacity: 0,
    ease: "hold",
  },
  {
    t: checkEnd(ITER_STARTS[1]) - 350,
    x: LOOP_VERDICT.x,
    y: LOOP_VERDICT.y,
    opacity: 1,
    ease: "linear",
  },
  {
    t: checkEnd(ITER_STARTS[1]) + 400,
    x: LOOP_VERDICT.x,
    y: LOOP_VERDICT.y,
    ease: "hold",
  },
  {
    t: checkEnd(ITER_STARTS[1]) + 650,
    x: LOOP_VERDICT.x,
    y: LOOP_VERDICT.y,
    opacity: 0,
    ease: "linear",
  },
]);

/** "yes → complete" flashes over the exit on the final verdict. */
const DONE_VERDICT_TRACK = defineTrack("cc-verdict-done", [
  {
    t: checkEnd(ITER_STARTS[2]) - 450,
    x: DONE_VERDICT.x,
    y: DONE_VERDICT.y,
    opacity: 0,
  },
  {
    t: checkEnd(ITER_STARTS[2]) - 300,
    x: DONE_VERDICT.x,
    y: DONE_VERDICT.y,
    opacity: 1,
    ease: "linear",
  },
  { t: 11500, x: DONE_VERDICT.x, y: DONE_VERDICT.y, ease: "hold" },
  {
    t: 11800,
    x: DONE_VERDICT.x,
    y: DONE_VERDICT.y,
    opacity: 0,
    ease: "linear",
  },
]);

// ─── Iteration readout (per-frame text, no React renders) ──────────────────

const clamp = (v: number, lo: number, hi: number) =>
  Math.min(hi, Math.max(lo, v));
const ramp = (t: number, a: number, b: number) =>
  clamp((t - a) / (b - a), 0, 1);

const iteration = (t: number) =>
  clamp(ITER_STARTS.filter((s) => t >= s).length, 1, ITERATIONS);

const readoutOpacity = (t: number) =>
  ramp(t, ITER_STARTS[0], ITER_STARTS[0] + 250) *
  (1 - ramp(t, FADE_START, FADE_END));

/** The counter that turns "it loops" into "it loops exactly until done". */
const IterationReadout = () => {
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
    const next = String(iteration(t));
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
      iteration{" "}
      <span ref={countRef} className={styles.readoutCount}>
        {iteration(posterTime)}
      </span>{" "}
      / {ITERATIONS}
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
    {/* Forward rail: parent → child run → check → complete */}
    <line
      x1={81}
      y1={PARENT_Y}
      x2={145}
      y2={CHILD_Y}
      className={styles.chromeDash}
      {...fine}
    />
    <line
      x1={167}
      y1={CHILD_Y}
      x2={222}
      y2={CHECK_Y}
      className={styles.chromeDash}
      {...fine}
    />
    <line
      x1={254}
      y1={CHECK_Y}
      x2={278}
      y2={DONE_Y}
      className={styles.chromeDash}
      {...fine}
    />
    {/* Child run slot */}
    <rect
      x={CHILD_X - 9}
      y={CHILD_Y - 9}
      width={18}
      height={18}
      className={styles.chromeSlot}
      {...fine}
      strokeDasharray="2 3"
    />
    {/* Termination check diamond */}
    <polygon
      points={`${CHECK_X},${CHECK_Y - 12} ${CHECK_X + 14},${CHECK_Y} ${CHECK_X},${CHECK_Y + 12} ${CHECK_X - 14},${CHECK_Y}`}
      className={styles.chromeCheck}
      {...stroke}
    />
    {/* Loop-back rail: check → down → left → up into the parent */}
    <path
      d={`M ${CHECK_X} ${CHECK_Y + 14} V ${LOOP_Y} H ${PARENT_X} V 103`}
      className={styles.chromeDash}
      {...fine}
    />
    <polygon
      points={`${PARENT_X},101 ${PARENT_X - 3.5},108 ${PARENT_X + 3.5},108`}
      className={styles.chromeFill}
    />
  </svg>
);

const StageLabel = ({
  x,
  y,
  muted = false,
  children,
}: {
  x: number;
  y: number;
  muted?: boolean;
  children: ReactNode;
}) => (
  <div
    className={`${styles.stageLabel} ${muted ? styles.labelMuted : ""}`}
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
  "Animated diagram of an agent reasoning loop: a durable parent spawns a child run of itself, the child runs, and the result reaches a termination check. On the first two iterations the verdict is to run again, and the token rides a loop-back rail into the parent to spawn the next iteration while a readout counts iterations one to three. On the third check the condition is met and the run exits to a complete box.";

export const ChildCycle = ({
  className,
  style,
}: {
  className?: string;
  style?: CSSProperties;
}) => (
  <div className={`${styles.wrap} ${className ?? ""}`} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.flowRoot}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={PARENT_X} y={58}>
          durable parent
        </StageLabel>
        <StageLabel x={CHILD_X} y={62}>
          child run
        </StageLabel>
        <StageLabel x={CHECK_X} y={58}>
          done?
        </StageLabel>
        <StageLabel x={DONE_X} y={102} muted>
          complete
        </StageLabel>
        <Flow.Token track={PARENT_TRACK}>
          <div className={styles.parentBox} />
        </Flow.Token>
        <Flow.Token track={DONE_TRACK}>
          <div className={styles.doneBox} />
        </Flow.Token>
        <Flow.Token track={LOOP_VERDICT_TRACK}>
          <div className={styles.verdictLoop}>no · run again</div>
        </Flow.Token>
        <Flow.Token track={DONE_VERDICT_TRACK}>
          <div className={styles.verdictDone}>yes → complete</div>
        </Flow.Token>
        <Flow.Token track={RUN_TRACK}>
          <div className={styles.square} />
        </Flow.Token>
        <IterationReadout />
      </Flow.Stage>
    </Flow.Root>
  </div>
);
