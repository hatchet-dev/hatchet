"use client";

import type { CSSProperties } from "react";
import { Flow, defineTrack, type FlowKeyframe } from "@/components/flow";
import styles from "./dagpipeline.module.css";

/**
 * Flow animation of one DAG run (docs.hatchet.run/v1/directed-acyclic-graphs).
 *
 * Five tasks in a diamond: A fans out to B and C, which both feed the join
 * task D, which feeds E. A run pulse enters from the left and travels the
 * edges; each node flips queued → running → done as the run reaches it. The
 * teaching beat is the join: B and C genuinely run concurrently, B finishes
 * early, and D visibly holds in a "waiting" state until C also completes —
 * a child task starts only after every parent has finished.
 *
 * Everything is scheduled at module scope from plain data; fully
 * deterministic; the loop wraps through an all-queued idle beat.
 */

// ─── Geometry (stage design units, 320 × 150 — landscape DAG) ──────────────

const STAGE_W = 320;
const STAGE_H = 150;

const NODE_W = 36;
const NODE_H = 20;

type XY = readonly [number, number];

/** Node centers. */
const A: XY = [36, 75];
const B: XY = [128, 40];
const C: XY = [128, 110];
const D: XY = [220, 75];
const E: XY = [292, 75];

/** Edge anchor points (node border midpoints). */
const A_R: XY = [A[0] + NODE_W / 2, A[1]];
const B_L: XY = [B[0] - NODE_W / 2, B[1]];
const B_R: XY = [B[0] + NODE_W / 2, B[1]];
const C_L: XY = [C[0] - NODE_W / 2, C[1]];
const C_R: XY = [C[0] + NODE_W / 2, C[1]];
const D_L: XY = [D[0] - NODE_W / 2, D[1]];
const D_R: XY = [D[0] + NODE_W / 2, D[1]];
const E_L: XY = [E[0] - NODE_W / 2, E[1]];

/** Trigger port feeding A from the stage edge. */
const PORT: XY = [6, 75];

// ─── Timeline (ms) ─────────────────────────────────────────────────────────
//
//    100  run pulse enters from the trigger port
//    400  A starts                     1400  A completes
//   1900  B and C start concurrently (fan-out)
//   2900  B completes — its output travels to D
//   3400  D flips to WAITING: one parent done, C still running
//   4400  C completes — the join is finally satisfied
//   4900  D starts                     5900  D completes
//   6400  E starts                     7400  E completes
//   8300  states fade back to queued; the loop wraps through idle

const START = 400;
const A_DONE = 1400;
const FAN_ARRIVE = 1900;
const B_DONE = 2900;
const B_AT_D = 3400;
const C_DONE = 4400;
const C_AT_D = 4900;
const D_DONE = 5900;
const D_AT_E = 6400;
const E_DONE = 7400;
const RESET = 8300;
const DURATION = 9200;

/** Static frame: B done, C running, D visibly holding for its second parent. */
const POSTER_TIME = 4000;

// ─── Tracks ────────────────────────────────────────────────────────────────

/** A stationary node that steps through run states. */
const nodeTrack = (id: string, [x, y]: XY, flips: [number, string][]) =>
  defineTrack(id, [
    { t: 0, x, y, state: "queued" },
    ...flips.map(([t, state]): FlowKeyframe => ({
      t,
      x,
      y,
      state,
      ease: "hold",
    })),
    { t: DURATION, x, y, ease: "hold" },
  ]);

const NODE_A = nodeTrack("dagp-a", A, [
  [START, "running"],
  [A_DONE, "done"],
  [RESET, "queued"],
]);
const NODE_B = nodeTrack("dagp-b", B, [
  [FAN_ARRIVE, "running"],
  [B_DONE, "done"],
  [RESET, "queued"],
]);
const NODE_C = nodeTrack("dagp-c", C, [
  [FAN_ARRIVE, "running"],
  [C_DONE, "done"],
  [RESET, "queued"],
]);
const NODE_D = nodeTrack("dagp-d", D, [
  [B_AT_D, "waiting"],
  [C_AT_D, "running"],
  [D_DONE, "done"],
  [RESET, "queued"],
]);
const NODE_E = nodeTrack("dagp-e", E, [
  [D_AT_E, "running"],
  [E_DONE, "done"],
  [RESET, "queued"],
]);

/** A run pulse traveling one edge: fade in en route, ease into the node. */
const travel = (
  id: string,
  depart: number,
  arrive: number,
  from: XY,
  mid: XY,
  to: XY,
) =>
  defineTrack(id, [
    { t: depart, x: from[0], y: from[1], opacity: 0, state: "pulse" },
    {
      t: depart + 140,
      x: from[0] + (mid[0] - from[0]) * 0.35,
      y: from[1] + (mid[1] - from[1]) * 0.35,
      opacity: 1,
      ease: "linear",
    },
    {
      t: Math.round((depart + arrive) / 2),
      x: mid[0],
      y: mid[1],
      ease: "linear",
    },
    { t: arrive, x: to[0], y: to[1], ease: "out" },
    { t: arrive + 120, x: to[0], y: to[1], opacity: 0, ease: "linear" },
  ]);

const PULSE_IN = travel(
  "dagp-in",
  100,
  START,
  PORT,
  [11, 75],
  [A[0] - NODE_W / 2, 75],
);
const PULSE_AB = travel("dagp-ab", A_DONE, FAN_ARRIVE, A_R, [82, 55], B_L);
const PULSE_AC = travel("dagp-ac", A_DONE, FAN_ARRIVE, A_R, [82, 95], C_L);
const PULSE_BD = travel("dagp-bd", B_DONE, B_AT_D, B_R, [174, 55], D_L);
const PULSE_CD = travel("dagp-cd", C_DONE, C_AT_D, C_R, [174, 95], D_L);
const PULSE_DE = travel("dagp-de", D_DONE, D_AT_E, D_R, [256, 75], E_L);

/** The join's annotation while it holds for its second parent. */
const WAIT_TAG = defineTrack("dagp-wait", [
  { t: B_AT_D + 150, x: D[0], y: 95, opacity: 0 },
  { t: B_AT_D + 300, x: D[0], y: 95, opacity: 1, ease: "linear" },
  { t: C_DONE, x: D[0], y: 95, ease: "hold" },
  { t: C_DONE + 200, x: D[0], y: 95, opacity: 0, ease: "linear" },
]);

// ─── Static chrome ─────────────────────────────────────────────────────────

const fine = {
  fill: "none",
  strokeWidth: 1.5,
  vectorEffect: "non-scaling-stroke",
} as const;

const edge = (from: XY, to: XY) => {
  const mx = (from[0] + to[0]) / 2;
  return `M ${from[0]} ${from[1]} C ${mx} ${from[1]}, ${mx} ${to[1]}, ${to[0]} ${to[1]}`;
};

const Chrome = () => (
  <svg
    viewBox={`0 0 ${STAGE_W} ${STAGE_H}`}
    aria-hidden="true"
    className={styles.chrome}
  >
    {/* Trigger port + inbound stub */}
    <rect
      x={PORT[0] - 2}
      y={PORT[1] - 2}
      width={4}
      height={4}
      className={styles.chromeFill}
    />
    <line
      x1={PORT[0] + 4}
      y1={PORT[1]}
      x2={A[0] - NODE_W / 2 - 2}
      y2={PORT[1]}
      className={styles.chromeEdge}
      {...fine}
    />
    {/* DAG edges */}
    <path d={edge(A_R, B_L)} className={styles.chromeEdge} {...fine} />
    <path d={edge(A_R, C_L)} className={styles.chromeEdge} {...fine} />
    <path d={edge(B_R, D_L)} className={styles.chromeEdge} {...fine} />
    <path d={edge(C_R, D_L)} className={styles.chromeEdge} {...fine} />
    <path d={edge(D_R, E_L)} className={styles.chromeEdge} {...fine} />
  </svg>
);

// ─── Tokens ────────────────────────────────────────────────────────────────

const Node = ({ label }: { label: string }) => (
  <div className={styles.node}>
    <span className={styles.nodeLabel}>{label}</span>
  </div>
);

const ARIA_LABEL =
  "Animated diagram of a DAG run: task A fans out to tasks B and C, which both feed task D, which feeds task E. A run pulse travels the edges and each task flips from queued to running to done. B and C run in parallel; B finishes first, and the join task D holds in a waiting state until C also completes, because a task starts only after all of its parents have finished.";

// ─── Export ────────────────────────────────────────────────────────────────

export const DagPipeline = ({
  className,
  style,
}: {
  className?: string;
  style?: CSSProperties;
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
        <div className={styles.parallelLabel}>parallel</div>
        <Flow.Token track={NODE_A}>
          <Node label="task a" />
        </Flow.Token>
        <Flow.Token track={NODE_B}>
          <Node label="task b" />
        </Flow.Token>
        <Flow.Token track={NODE_C}>
          <Node label="task c" />
        </Flow.Token>
        <Flow.Token track={NODE_D}>
          <Node label="task d" />
        </Flow.Token>
        <Flow.Token track={NODE_E}>
          <Node label="task e" />
        </Flow.Token>
        <Flow.Token track={WAIT_TAG}>
          <div className={styles.waitTag}>waiting on task c</div>
        </Flow.Token>
        <Flow.Token track={PULSE_IN}>
          <div className={styles.pulse} />
        </Flow.Token>
        <Flow.Token track={PULSE_AB}>
          <div className={styles.pulse} />
        </Flow.Token>
        <Flow.Token track={PULSE_AC}>
          <div className={styles.pulse} />
        </Flow.Token>
        <Flow.Token track={PULSE_BD}>
          <div className={styles.pulse} />
        </Flow.Token>
        <Flow.Token track={PULSE_CD}>
          <div className={styles.pulse} />
        </Flow.Token>
        <Flow.Token track={PULSE_DE}>
          <div className={styles.pulse} />
        </Flow.Token>
      </Flow.Stage>
    </Flow.Root>
  </div>
);
