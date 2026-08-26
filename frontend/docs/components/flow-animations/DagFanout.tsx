"use client";

import type { CSSProperties } from "react";
import { Flow, defineTrack, type FlowKeyframe } from "@/components/flow";
import styles from "./dagfanout.module.css";

/**
 * Flow animation of DAG fan-out / fan-in
 * (docs.hatchet.run/v1/directed-acyclic-graphs).
 *
 * One trigger fires three sibling tasks at once. The branches genuinely run
 * concurrently and finish at different times; as each completes, its output
 * travels the converging edge and docks in a slot beside the result task.
 * The result stays a dashed outline — waiting — until the third output
 * docks, and only then does it materialize: outputs from every parent are
 * required before the downstream task can run.
 *
 * Everything is scheduled at module scope from plain data; fully
 * deterministic; the loop wraps through an all-queued idle beat.
 */

// ─── Geometry (stage design units, 320 × 160 — landscape DAG) ──────────────

const STAGE_W = 320;
const STAGE_H = 160;

const NODE_W = 40;
const NODE_H = 22;

type XY = readonly [number, number];

/** Trigger port on the left, three parallel tasks, one result on the right. */
const TRIGGER: XY = [28, 80];
const TASKS: readonly XY[] = [
  [142, 32],
  [142, 80],
  [142, 128],
];
const RESULT: XY = [276, 80];
const RESULT_W = 46;
const RESULT_H = 26;

/** Converging edges aim at a merge point just left of the dock column. */
const MERGE: XY = [238, 80];

/** Output dock: one slot per branch, beside the result, filled by arrival. */
const DOCK_X = 246;
const DOCK_YS = [70, 80, 90] as const;

// ─── Timeline (ms) ─────────────────────────────────────────────────────────
//
//    300  the trigger fires; three run pulses fan out concurrently
//    900  task 1, task 2, task 3 all start
//   2300  task 1 completes → its output docks at the result at 2800
//   2900  task 3 completes → output docks at 3400 (result still waiting)
//   3600  task 2 completes → output docks at 4100: all parents done
//   4100  the result materializes; docked outputs are absorbed
//   6200  states fade back to queued; the loop wraps through idle

const FIRE = 300;
const TASKS_START = 900;
const DOCK_TRAVEL = 500;
/** Per-branch completion times — staggered so the fan-in reads. */
const TASK_DONE = [2300, 3600, 2900] as const;
/** Dock slots fill in order of completion: task 1, task 3, task 2. */
const DOCK_SLOT = [0, 2, 1] as const;
const ALL_DONE = Math.max(...TASK_DONE) + DOCK_TRAVEL; // 4100
const RESET = 6200;
const DURATION = 7200;

/** Static frame: one output docked, one mid-flight, one branch still running. */
const POSTER_TIME = 3200;

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

const TRIGGER_TRACK = nodeTrack("dagf-trigger", TRIGGER, [
  [FIRE, "fired"],
  [TASKS_START, "idle"],
]);

const TASK_TRACKS = TASKS.map((xy, i) =>
  nodeTrack(`dagf-task-${i}`, xy, [
    [TASKS_START, "running"],
    [TASK_DONE[i], "done"],
    [RESET, "queued"],
  ]),
);

const RESULT_TRACK = nodeTrack("dagf-result", RESULT, [
  [TASK_DONE[0] + DOCK_TRAVEL, "waiting"],
  [ALL_DONE, "done"],
  [RESET, "queued"],
]);

/** A run pulse traveling one edge: fade in en route, ease into the target. */
const travel = (
  id: string,
  depart: number,
  arrive: number,
  from: XY,
  mid: XY,
  to: XY,
  tail: FlowKeyframe[],
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
    ...tail,
  ]);

/** Trigger → task pulses, fanning out concurrently. */
const FAN_TRACKS = TASKS.map(([tx, ty], i) =>
  travel(
    `dagf-fan-${i}`,
    FIRE,
    TASKS_START,
    [TRIGGER[0] + 6, TRIGGER[1]],
    [86, (TRIGGER[1] + ty) / 2],
    [tx - NODE_W / 2, ty],
    [
      {
        t: TASKS_START + 120,
        x: tx - NODE_W / 2,
        y: ty,
        opacity: 0,
        ease: "linear",
      },
    ],
  ),
);

/** Task → result outputs: dock beside the result, absorbed once it runs. */
const OUTPUT_TRACKS = TASKS.map(([tx, ty], i) => {
  const depart = TASK_DONE[i];
  const dockY = DOCK_YS[DOCK_SLOT[i]];
  return travel(
    `dagf-out-${i}`,
    depart,
    depart + DOCK_TRAVEL,
    [tx + NODE_W / 2, ty],
    [(tx + NODE_W / 2 + MERGE[0]) / 2, (ty + MERGE[1]) / 2],
    [DOCK_X, dockY],
    [
      {
        t: depart + DOCK_TRAVEL + 40,
        x: DOCK_X,
        y: dockY,
        ease: "hold",
        state: "docked",
      },
      { t: ALL_DONE + 200, x: DOCK_X, y: dockY, ease: "hold" },
      {
        t: ALL_DONE + 550,
        x: RESULT[0] - RESULT_W / 2 + 6,
        y: RESULT[1],
        opacity: 0,
        scale: 0.5,
        ease: "in",
      },
    ],
  );
});

/** Annotation while the result holds for its remaining parents. */
const WAIT_TAG = defineTrack("dagf-wait", [
  { t: TASK_DONE[0] + DOCK_TRAVEL + 150, x: 268, y: 102, opacity: 0 },
  {
    t: TASK_DONE[0] + DOCK_TRAVEL + 300,
    x: 268,
    y: 102,
    opacity: 1,
    ease: "linear",
  },
  { t: ALL_DONE, x: 268, y: 102, ease: "hold" },
  { t: ALL_DONE + 200, x: 268, y: 102, opacity: 0, ease: "linear" },
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
    {/* Fan-out: trigger → each parallel task */}
    {TASKS.map(([tx, ty]) => (
      <path
        key={`fan-${ty}`}
        d={edge([TRIGGER[0] + 6, TRIGGER[1]], [tx - NODE_W / 2, ty])}
        className={styles.chromeEdge}
        {...fine}
      />
    ))}
    {/* Fan-in: each task → the merge point at the result's dock */}
    {TASKS.map(([tx, ty]) => (
      <path
        key={`join-${ty}`}
        d={edge([tx + NODE_W / 2, ty], MERGE)}
        className={styles.chromeEdge}
        {...fine}
      />
    ))}
  </svg>
);

// ─── Tokens ────────────────────────────────────────────────────────────────

const ARIA_LABEL =
  "Animated diagram of DAG fan-out: a trigger fires three sibling tasks at once. The tasks run in parallel and finish at different times; as each completes, its output travels a converging edge and docks beside the result task. The result stays a dashed waiting outline until all three outputs have arrived, then materializes — a downstream task runs only after every parallel branch has completed.";

// ─── Export ────────────────────────────────────────────────────────────────

export const DagFanout = ({
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
        <div className={styles.triggerLabel}>trigger</div>
        <Flow.Token track={TRIGGER_TRACK}>
          <div className={styles.triggerPort} />
        </Flow.Token>
        {TASK_TRACKS.map((track, i) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.node}>
              <span className={styles.nodeLabel}>task {i + 1}</span>
            </div>
          </Flow.Token>
        ))}
        <Flow.Token track={RESULT_TRACK}>
          <div className={styles.result}>
            <span className={styles.nodeLabel}>result</span>
          </div>
        </Flow.Token>
        <Flow.Token track={WAIT_TAG}>
          <div className={styles.waitTag}>waiting on parents</div>
        </Flow.Token>
        {FAN_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.pulse} />
          </Flow.Token>
        ))}
        {OUTPUT_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.pulse} />
          </Flow.Token>
        ))}
      </Flow.Stage>
    </Flow.Root>
  </div>
);
