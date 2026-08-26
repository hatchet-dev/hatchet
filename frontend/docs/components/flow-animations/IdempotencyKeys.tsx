"use client";

import type { CSSProperties, ReactNode } from "react";
import { Flow, defineTrack, type FlowKeyframe } from "@/components/flow";
import { Text } from "@/components/flow/Text";
import styles from "./idempotencykeys.module.css";

/**
 * Flow animation of Hatchet's idempotency keys
 * (docs.hatchet.run/v1/idempotency).
 *
 * Three trigger sources (API / WEBHOOK / RETRY) emit tasks toward a key gate.
 * Each trigger's input evaluates to an idempotency key (the `k:…` tag riding
 * under the square). The first task to arrive claims the key — the register
 * slot above the gate fills — and runs on the worker. While the key is held,
 * later triggers with the same key are dropped at the gate (by design, not a
 * failure — so dimmed fg with a slash, never magenta), while a task with a
 * different key passes straight through and runs concurrently. When the
 * claiming run reaches a terminal status the key is released, and the next
 * same-key trigger claims it again — closing the loop.
 *
 * One run per key while the key is held; everything seeded/deterministic.
 */

// ─── Geometry (stage design units, 320 × 208 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 208;

const SPAWN_X = 30;
const MERGE_X = 118;
const TRACK_Y = 120;
// Tasks pause here awaiting the gate's verdict — far enough left of GATE_X
// that the key tag underneath never touches the gate bars.
const CHECK_X = 140;
const GATE_X = 152;

const SRC_Y = { api: 60, webhook: 120, retry: 180 } as const;

/** Key register slot above the gate. */
const SLOT_X = 152;
const SLOT_Y = 46;
// Held-key / released tags sit right of the slot box (right edge ≈162):
// their LEFT edge anchors here (see .sideTag translateX), so neither can
// reach back over the box regardless of text width.
const HELD_X = 168;

const WORKER_X = 252;
const RUN_Y1 = 108; // two run positions inside the worker box
const RUN_Y2 = 132;

const DROP_LABEL_Y = 154;

// ─── Timeline ──────────────────────────────────────────────────────────────
//
// Authored at the original 8200ms base pace, then uniformly stretched by
// SCALE so relative pacing is preserved while every beat gets room to read.
// Base-time storyboard:
//
//    200  A  API      k:7f2a  claims the key at 1250, runs 1650 → 4300
//   1500  B  WEBHOOK  k:7f2a  collides at 2550 → dropped (lingers, sinks)
//   2400  C  API      k:31c9  different key: passes, runs 3800 → 5100
//   3000  D  RETRY    k:7f2a  collides at 4050 → dropped
//   4500           A's key released (terminal status reached at 4300)
//   4400  E  WEBHOOK  k:7f2a  key is free again: claims at 5450, runs
//   7400           E's key released; the loop wraps empty → seamless

const SCALE = 1.8;
const at = (t: number) => Math.round(t * SCALE);
/** Stretch a base-paced keyframe list onto the slowed loop clock. */
const scaled = (keyframes: FlowKeyframe[]): FlowKeyframe[] =>
  keyframes.map((k) => ({ ...k, t: at(k.t) }));

const DURATION = at(8200); // 14760
/** Static frame: A processing (blue), key held, B dropped, C in flight. */
const POSTER_TIME = at(2700); // 4860

/** Spawn at a source, fade in, converge to the shared track, pause at the gate. */
const approach = (
  spawn: number,
  srcY: number,
  fadeY: number,
  gateAt: number
): FlowKeyframe[] => [
  { t: spawn, x: SPAWN_X, y: srcY, opacity: 0, state: "inflight" },
  { t: spawn + 180, x: SPAWN_X + 16, y: fadeY, opacity: 1, ease: "linear" },
  { t: gateAt - 300, x: MERGE_X, y: TRACK_Y, ease: "inOut" },
  { t: gateAt, x: CHECK_X, y: TRACK_Y, ease: "out", state: "checking" },
];

/** Idempotency collision: dim, slash tick, linger so it reads, sink out. */
// Note: the drop must fully clear CHECK_X by decideAt+650 — the next task
// (C at base 3200) pauses on the same spot.
const drop = (decideAt: number): FlowKeyframe[] => [
  { t: decideAt, x: CHECK_X, y: TRACK_Y, ease: "hold", state: "dropped" },
  { t: decideAt + 300, x: CHECK_X, y: TRACK_Y, ease: "hold" },
  { t: decideAt + 650, x: CHECK_X, y: TRACK_Y + 9, opacity: 0, ease: "in" },
];

/** Pass the gate, process at the worker (blue), reach terminal status, fade. */
const run = (departGate: number, runY: number, procEnd: number): FlowKeyframe[] => [
  { t: departGate, x: CHECK_X, y: TRACK_Y, ease: "hold" },
  { t: departGate + 400, x: WORKER_X, y: runY, ease: "inOut", state: "active" },
  { t: procEnd, x: WORKER_X, y: runY, ease: "hold", state: "done" },
  { t: procEnd + 300, x: WORKER_X, y: runY, ease: "hold" },
  { t: procEnd + 600, x: WORKER_X, y: runY, opacity: 0, scale: 1.5, ease: "out" },
];

const TASK_A = defineTrack(
  "idem-a",
  scaled([...approach(200, SRC_Y.api, 67, 1000), ...run(1250, RUN_Y1, 4300)])
);
const TASK_B = defineTrack(
  "idem-b",
  scaled([...approach(1500, SRC_Y.webhook, SRC_Y.webhook, 2300), ...drop(2550)])
);
const TASK_C = defineTrack(
  "idem-c",
  scaled([...approach(2400, SRC_Y.api, 67, 3200), ...run(3400, RUN_Y2, 5100)])
);
const TASK_D = defineTrack(
  "idem-d",
  scaled([...approach(3000, SRC_Y.retry, 173, 3800), ...drop(4050)])
);
const TASK_E = defineTrack(
  "idem-e",
  scaled([...approach(4400, SRC_Y.webhook, SRC_Y.webhook, 5200), ...run(5450, RUN_Y1, 7200)])
);

/** The register slot's occupant: claimed on arrival, released on terminal status. */
const SLOT_FILL = defineTrack(
  "idem-slot",
  scaled([
    { t: 1100, x: SLOT_X, y: SLOT_Y, opacity: 0 },
    { t: 1250, x: SLOT_X, y: SLOT_Y, opacity: 1, ease: "out", state: "claim" },
    { t: 4500, x: SLOT_X, y: SLOT_Y, ease: "hold", state: "released" },
    { t: 4700, x: SLOT_X, y: SLOT_Y, opacity: 0, scale: 1.6, ease: "out" },
    { t: 5300, x: SLOT_X, y: SLOT_Y, opacity: 0, scale: 1, ease: "hold" },
    { t: 5450, x: SLOT_X, y: SLOT_Y, opacity: 1, ease: "out", state: "claim" },
    { t: 7400, x: SLOT_X, y: SLOT_Y, ease: "hold", state: "released" },
    { t: 7600, x: SLOT_X, y: SLOT_Y, opacity: 0, scale: 1.6, ease: "out" },
  ])
);

/** The held key's value, shown beside the slot while it's claimed. */
const HELD_TAG = defineTrack(
  "idem-held",
  scaled([
    { t: 1250, x: HELD_X, y: SLOT_Y, opacity: 0 },
    { t: 1400, x: HELD_X, y: SLOT_Y, opacity: 1, ease: "linear" },
    { t: 4450, x: HELD_X, y: SLOT_Y, ease: "hold" },
    { t: 4600, x: HELD_X, y: SLOT_Y, opacity: 0, ease: "linear" },
    { t: 5450, x: HELD_X, y: SLOT_Y, opacity: 0, ease: "hold" },
    { t: 5600, x: HELD_X, y: SLOT_Y, opacity: 1, ease: "linear" },
    { t: 7350, x: HELD_X, y: SLOT_Y, ease: "hold" },
    { t: 7500, x: HELD_X, y: SLOT_Y, opacity: 0, ease: "linear" },
  ])
);

/** Release beat, flashed as the slot empties — held long enough to read. */
const RELEASED_TAG = defineTrack(
  "idem-released",
  scaled([
    { t: 4600, x: HELD_X, y: SLOT_Y, opacity: 0 },
    { t: 4750, x: HELD_X, y: SLOT_Y, opacity: 1, ease: "linear" },
    { t: 5150, x: HELD_X, y: SLOT_Y, ease: "hold" },
    { t: 5300, x: HELD_X, y: SLOT_Y, opacity: 0, ease: "linear" },
    { t: 7600, x: HELD_X, y: SLOT_Y, opacity: 0, ease: "hold" },
    { t: 7750, x: HELD_X, y: SLOT_Y, opacity: 1, ease: "linear" },
    { t: 8050, x: HELD_X, y: SLOT_Y, ease: "hold" },
    { t: 8200, x: HELD_X, y: SLOT_Y, opacity: 0, ease: "linear" },
  ])
);

/** Collision annotation under the gate, once per dropped task. */
const DROP_LABEL = defineTrack(
  "idem-drop-label",
  scaled([
    { t: 2550, x: GATE_X, y: DROP_LABEL_Y, opacity: 0 },
    { t: 2700, x: GATE_X, y: DROP_LABEL_Y, opacity: 1, ease: "linear" },
    { t: 3250, x: GATE_X, y: DROP_LABEL_Y, ease: "hold" },
    { t: 3500, x: GATE_X, y: DROP_LABEL_Y, opacity: 0, ease: "linear" },
    { t: 4050, x: GATE_X, y: DROP_LABEL_Y, opacity: 0, ease: "hold" },
    { t: 4200, x: GATE_X, y: DROP_LABEL_Y, opacity: 1, ease: "linear" },
    { t: 4700, x: GATE_X, y: DROP_LABEL_Y, ease: "hold" },
    { t: 4950, x: GATE_X, y: DROP_LABEL_Y, opacity: 0, ease: "linear" },
  ])
);

/** Worker box border brightens while any run is processing. */
const WORKER = defineTrack(
  "idem-worker",
  scaled([
    { t: 0, x: WORKER_X, y: TRACK_Y, state: "idle" },
    { t: 1650, x: WORKER_X, y: TRACK_Y, state: "busy", ease: "hold" },
    { t: 5100, x: WORKER_X, y: TRACK_Y, state: "idle", ease: "hold" },
    { t: 5850, x: WORKER_X, y: TRACK_Y, state: "busy", ease: "hold" },
    { t: 7200, x: WORKER_X, y: TRACK_Y, state: "idle", ease: "hold" },
    { t: 8200, x: WORKER_X, y: TRACK_Y, ease: "hold" },
  ])
);

// ─── Static chrome ─────────────────────────────────────────────────────────

const stroke = { fill: "none", strokeWidth: 2, vectorEffect: "non-scaling-stroke" } as const;
const fine = { fill: "none", strokeWidth: 1.5, vectorEffect: "non-scaling-stroke" } as const;

const Chrome = () => (
  <svg viewBox={`0 0 ${STAGE_W} ${STAGE_H}`} aria-hidden="true">
    {/* Trigger source markers */}
    {Object.values(SRC_Y).map((y) => (
      <rect key={y} x={28} y={y - 2} width={4} height={4} className={styles.chromeFill} />
    ))}
    {/* Converging trigger tracks */}
    <line x1={36} y1={SRC_Y.api} x2={114} y2={117} className={styles.chromeDash} {...fine} />
    <line x1={36} y1={SRC_Y.webhook} x2={114} y2={120} className={styles.chromeDash} {...fine} />
    <line x1={36} y1={SRC_Y.retry} x2={114} y2={123} className={styles.chromeDash} {...fine} />
    <line x1={MERGE_X} y1={TRACK_Y} x2={134} y2={TRACK_Y} className={styles.chromeDash} {...fine} />
    {/* The key gate: a gap in a vertical barrier on the track */}
    <line x1={GATE_X} y1={102} x2={GATE_X} y2={114} className={styles.chromeStroke} {...stroke} />
    <line x1={GATE_X} y1={126} x2={GATE_X} y2={138} className={styles.chromeStroke} {...stroke} />
    {/* Key register slot + its tie to the gate */}
    <rect
      x={SLOT_X - 6}
      y={SLOT_Y - 6}
      width={12}
      height={12}
      className={styles.chromeSlot}
      {...fine}
      strokeDasharray="2 3"
    />
    <line x1={GATE_X} y1={SLOT_Y + 8} x2={GATE_X} y2={100} className={styles.chromeDash} {...fine} />
    {/* Gate → worker connector */}
    <line x1={158} y1={TRACK_Y} x2={230} y2={TRACK_Y} className={styles.chromeDash} {...fine} />
  </svg>
);

const StageLabel = ({
  x,
  y,
  children,
}: {
  x: number;
  y: number;
  children: ReactNode;
}) => (
  <div
    className={styles.stageLabel}
    style={{
      left: `calc(var(--flow-u) * ${x})`,
      top: `calc(var(--flow-u) * ${y})`,
    }}
  >
    {children}
  </div>
);

// ─── Tokens ────────────────────────────────────────────────────────────────

/** A task square carrying its computed idempotency key. */
const Task = ({ k, alt = false }: { k: string; alt?: boolean }) => (
  <div className={styles.task}>
    <div className={`${styles.square} ${alt ? styles.squareAlt : ""}`} />
    <div className={styles.keyTag}>{k}</div>
  </div>
);

const KEY_A = "k:7f2a";
const KEY_B = "k:31c9";

const CAPTION =
  "The first trigger claims the idempotency key. Duplicates are dropped until the key is released.";

const ARIA_LABEL =
  "Animated diagram of Hatchet idempotency keys: triggers from an API, a webhook, and a retry emit tasks toward a key gate. The first task claims the idempotency key and runs on a worker; duplicate triggers with the same key are dropped at the gate, while a task with a different key runs normally. When the run completes, the key is released and the next same-key trigger claims it and runs.";

// ─── Export ────────────────────────────────────────────────────────────────

export const IdempotencyKeys = ({
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
        <StageLabel x={SPAWN_X} y={SRC_Y.api - 13}>
          api
        </StageLabel>
        <StageLabel x={SPAWN_X} y={SRC_Y.webhook - 13}>
          webhook
        </StageLabel>
        <StageLabel x={SPAWN_X} y={SRC_Y.retry - 13}>
          retry
        </StageLabel>
        <StageLabel x={SLOT_X} y={20}>
          idempotency key
        </StageLabel>
        <StageLabel x={WORKER_X} y={80}>
          worker
        </StageLabel>
        <Flow.Token track={WORKER}>
          <div className={styles.workerBox} />
        </Flow.Token>
        <Flow.Token track={SLOT_FILL}>
          <div className={styles.slotFill} />
        </Flow.Token>
        <Flow.Token track={HELD_TAG}>
          <div className={styles.heldTag}>{KEY_A}</div>
        </Flow.Token>
        <Flow.Token track={RELEASED_TAG}>
          <div className={styles.releasedTag}>released</div>
        </Flow.Token>
        <Flow.Token track={DROP_LABEL}>
          <div className={styles.dropLabel}>same key · dropped</div>
        </Flow.Token>
        <Flow.Token track={TASK_A}>
          <Task k={KEY_A} />
        </Flow.Token>
        <Flow.Token track={TASK_B}>
          <Task k={KEY_A} />
        </Flow.Token>
        <Flow.Token track={TASK_C}>
          <Task k={KEY_B} alt />
        </Flow.Token>
        <Flow.Token track={TASK_D}>
          <Task k={KEY_A} />
        </Flow.Token>
        <Flow.Token track={TASK_E}>
          <Task k={KEY_A} />
        </Flow.Token>
      </Flow.Stage>
    </Flow.Root>
    {showCaption && (
      <Text.Small as="p" secondary balance className={styles.caption}>
        {CAPTION}
      </Text.Small>
    )}
  </div>
);
