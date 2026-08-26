"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  createSeededRandom,
  defineTrack,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import { Text } from "@/components/flow/Text";
import styles from "./webhookingest.module.css";

/**
 * Flow animation of Hatchet's built-in webhook ingestors
 * (docs.hatchet.run/v1/webhooks).
 *
 * Three senders (Stripe, GitHub, Slack) post webhook requests that converge on
 * a Hatchet endpoint. Each request passes a signature-validation gate — one
 * forged request flickers and is dropped, dimmed — then a CEL event-key
 * expression derives a routable key from the payload (`'stripe:' + input.type`
 * → `stripe:invoice.paid`, shown morphing in a mono readout). The keyed event
 * lands in the events strip, where a small run indicator fires for the
 * subscribed workflow. Five webhooks per ~14.6s loop; landed events and run
 * marks fade in the final beat so the wrap is seamless.
 */

// ─── Geometry (stage design units, 320 × 208 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 208;

/** The pipeline axis every webhook travels along. */
const AXIS = 76;

/** Sender chips, stacked at the left; the middle chip sits on the axis. */
const CHIP_Y = [40, 76, 112] as const;
const CHIP_RIGHT = 54;
const ENTRY_X = 66;
const ELBOW_X = 84;

/** Signature-validation gate (dashed vertical line + lock glyph). */
const GATE_X = 124;
const GATE_Y0 = 52;
const GATE_Y1 = 100;

/** CEL key stage (dashed transform box on the axis). */
const KEY_BOX = { x: 168, y: 64, w: 28, h: 24 };
const KEY_X = KEY_BOX.x + KEY_BOX.w / 2;
const EXIT_WP = 222;

/** Events strip: four landing rows, each with a run indicator to its right. */
const STRIP = { x: 240, y: 36, w: 64, h: 104 };
const ROW_Y = [48, 74, 100, 126] as const;
const LAND_X = 252;
const RUN_X = 269;

/** Derivation readout (payload → key morph) and the rejection note. */
const READOUT = { x: 160, y: 172 };
const NOTE = { x: 124, y: 118 };

const LABEL_Y = 16;

// ─── Webhook schedule (seeded, deterministic) ──────────────────────────────

const SOURCES = ["Stripe", "GitHub", "Slack"] as const;

interface HookSpec {
  source: 0 | 1 | 2;
  base: number;
  invalid?: boolean;
  expr?: string;
  payload?: string;
  key?: string;
}

/** Expressions and keys follow the docs' examples (docs.hatchet.run/v1/webhooks). */
const HOOK_SPECS: HookSpec[] = [
  {
    source: 0,
    base: 300,
    expr: "'stripe:' + input.type",
    payload: '{ "type": "invoice.paid" }',
    key: "stripe:invoice.paid",
  },
  {
    source: 1,
    base: 2800,
    expr: "'github:' + headers['x-github-event']",
    payload: "x-github-event: push",
    key: "github:push",
  },
  {
    source: 2,
    base: 5300,
    expr: "'slack:' + input.event.type",
    payload: '{ "event": { "type": "app_mention" } }',
    key: "slack:app_mention",
  },
  // A forged request: bounces off the validation gate, no key derived.
  { source: 1, base: 7800, invalid: true },
  {
    source: 0,
    base: 9800,
    expr: "'stripe:' + input.type",
    payload: '{ "type": "payment_intent.created" }',
    key: "stripe:payment_intent.created",
  },
];

/** Fixed per-webhook milestones, jittered by the seeded PRNG. */
const HOOKS = (() => {
  const rng = createSeededRandom("webhook-ingest");
  let row = 0;
  return HOOK_SPECS.map((spec) => {
    const spawn = spec.base + Math.round(rng() * 240 - 120);
    const dwell = 280 + Math.round(rng() * 140); // pause at the gate
    const gateAt = spawn + 1020;
    const verdictAt = gateAt + dwell;
    const keyAt = verdictAt + 380; // arrives in the key box
    const keyedAt = keyAt + 400; // CEL derivation done
    const exitAt = keyedAt + 210;
    const landAt = exitAt + 380; // lands in its events row
    return {
      ...spec,
      spawn,
      gateAt,
      verdictAt,
      keyAt,
      keyedAt,
      exitAt,
      landAt,
      row: spec.invalid ? -1 : row++,
    };
  });
})();

type Hook = (typeof HOOKS)[number];

const VALID_HOOKS = HOOKS.filter((h) => !h.invalid);
const INVALID_HOOK = HOOKS.find((h) => h.invalid) as Hook;

// ─── Loop bookkeeping ──────────────────────────────────────────────────────

const LAST_LAND = Math.max(...VALID_HOOKS.map((h) => h.landAt));
/** Run-indicator settle + a breather, then the fade-out wrap beat. */
const DURATION = Math.ceil((LAST_LAND + 2050) / 200) * 200;
const FADE_AT = DURATION - 520;
/**
 * Static frame: the Slack webhook sits keyed (accent) in the CEL box with the
 * readout showing its derived key, while two earlier events rest in the strip
 * with fired run indicators.
 */
const POSTER_TIME = HOOKS[2].keyedAt + 80;

// ─── Tracks ────────────────────────────────────────────────────────────────

/** Chip port → elbow → bus → gate → key box → events row (or gate bounce). */
const hookTrack = (h: Hook, i: number): FlowTrack => {
  const cy = CHIP_Y[h.source];
  const kfs: FlowKeyframe[] = [
    { t: h.spawn, x: ENTRY_X, y: cy, opacity: 0, state: "raw" },
    { t: h.spawn + 140, x: ENTRY_X + 6, y: cy, opacity: 1, ease: "linear" },
  ];
  if (cy === AXIS) {
    kfs.push({ t: h.spawn + 620, x: ELBOW_X + 10, y: AXIS, ease: "inOut" });
  } else {
    kfs.push(
      { t: h.spawn + 430, x: ELBOW_X, y: cy, ease: "inOut" },
      { t: h.spawn + 700, x: ELBOW_X, y: AXIS, ease: "inOut" }
    );
  }
  kfs.push({ t: h.gateAt, x: GATE_X, y: AXIS, ease: "out", state: "checking" });
  if (h.invalid) {
    // Signature check fails: flicker at the gate, then a dimmed drop.
    kfs.push(
      { t: h.verdictAt, x: GATE_X, y: AXIS, ease: "hold", state: "rejected" },
      { t: h.verdictAt + 560, x: GATE_X, y: AXIS, ease: "hold" },
      { t: h.verdictAt + 1120, x: GATE_X, y: AXIS + 22, opacity: 0, ease: "in" }
    );
    return defineTrack(`hook-${i}`, kfs);
  }
  kfs.push(
    { t: h.verdictAt, x: GATE_X, y: AXIS, ease: "hold", state: "valid" },
    { t: h.keyAt, x: KEY_X, y: AXIS, ease: "inOut", state: "keying" },
    { t: h.keyedAt, x: KEY_X, y: AXIS, ease: "hold", state: "keyed" },
    { t: h.exitAt, x: EXIT_WP, y: AXIS, ease: "in" },
    { t: h.landAt, x: LAND_X, y: ROW_Y[h.row], ease: "out", state: "stored" },
    { t: FADE_AT, x: LAND_X, y: ROW_Y[h.row], ease: "hold" },
    { t: DURATION, x: LAND_X, y: ROW_Y[h.row], opacity: 0, ease: "linear" }
  );
  return defineTrack(`hook-${i}`, kfs);
};

const HOOK_TRACKS = HOOKS.map(hookTrack);

/** The run indicator beside a landed event: flashes on, settles, persists. */
const runTrack = (h: Hook): FlowTrack => {
  const y = ROW_Y[h.row];
  return defineTrack(`run-${h.row}`, [
    { t: h.landAt + 80, x: RUN_X, y, opacity: 0, state: "armed" },
    { t: h.landAt + 220, x: RUN_X, y, opacity: 1, ease: "linear", state: "firing" },
    { t: h.landAt + 920, x: RUN_X, y, ease: "hold", state: "fired" },
    { t: FADE_AT, x: RUN_X, y, ease: "hold" },
    { t: DURATION, x: RUN_X, y, opacity: 0, ease: "linear" },
  ]);
};

const RUN_TRACKS = VALID_HOOKS.map(runTrack);

/** The derivation readout: payload text crossfades to the derived key. */
const readoutTrack = (h: Hook, i: number): FlowTrack =>
  defineTrack(`readout-${i}`, [
    { t: h.gateAt + 150, x: READOUT.x, y: READOUT.y, opacity: 0, state: "payload" },
    { t: h.gateAt + 400, x: READOUT.x, y: READOUT.y, opacity: 1, ease: "linear" },
    { t: h.keyedAt, x: READOUT.x, y: READOUT.y, ease: "hold", state: "keyed" },
    { t: h.landAt + 500, x: READOUT.x, y: READOUT.y, ease: "hold" },
    { t: h.landAt + 850, x: READOUT.x, y: READOUT.y, opacity: 0, ease: "linear" },
  ]);

const READOUT_TRACKS = VALID_HOOKS.map(readoutTrack);

/** The rejection note under the gate. */
const NOTE_TRACK: FlowTrack = (() => {
  const v = INVALID_HOOK.verdictAt;
  return defineTrack("note", [
    { t: v + 150, x: NOTE.x, y: NOTE.y, opacity: 0 },
    { t: v + 400, x: NOTE.x, y: NOTE.y, opacity: 1, ease: "linear" },
    { t: v + 1500, x: NOTE.x, y: NOTE.y, ease: "hold" },
    { t: v + 1900, x: NOTE.x, y: NOTE.y, opacity: 0, ease: "linear" },
  ]);
})();

// ─── Static chrome ─────────────────────────────────────────────────────────

const fine = { fill: "none", strokeWidth: 1, vectorEffect: "non-scaling-stroke" } as const;

const Chrome = () => (
  <svg viewBox={`0 0 ${STAGE_W} ${STAGE_H}`} aria-hidden="true">
    {/* Sender ports + converging connectors */}
    {CHIP_Y.map((cy) => (
      <rect key={cy} x={59} y={cy - 1.5} width={3} height={3} className={styles.chromeFill} />
    ))}
    <path d={`M 64 ${CHIP_Y[0]} H ${ELBOW_X} V ${AXIS}`} className={styles.chromeDash} {...fine} />
    <path d={`M 64 ${CHIP_Y[2]} H ${ELBOW_X} V ${AXIS}`} className={styles.chromeDash} {...fine} />
    <line x1={64} y1={AXIS} x2={GATE_X - 6} y2={AXIS} className={styles.chromeDash} {...fine} />
    {/* Validation gate: dashed line + lock glyph */}
    <line x1={GATE_X} y1={GATE_Y0} x2={GATE_X} y2={GATE_Y1} className={styles.chromeDash} {...fine} />
    <rect x={GATE_X - 1.5} y={GATE_Y0 - 1.5} width={3} height={3} className={styles.chromeFill} />
    <rect x={GATE_X - 1.5} y={GATE_Y1 - 1.5} width={3} height={3} className={styles.chromeFill} />
    <path
      d={`M ${GATE_X - 3.2} 44 v -1.6 a 3.2 3.2 0 0 1 6.4 0 V 44`}
      className={styles.chromeStroke}
      {...fine}
    />
    <rect
      x={GATE_X - 4.5}
      y={44}
      width={9}
      height={6.5}
      className={styles.chromeStroke}
      {...fine}
    />
    <rect x={GATE_X - 0.8} y={46.4} width={1.6} height={1.6} className={styles.chromeFill} />
    {/* Gate → key box connector */}
    <line x1={GATE_X + 5} y1={AXIS} x2={KEY_BOX.x - 2} y2={AXIS} className={styles.chromeDash} {...fine} />
    {/* CEL key stage: dashed transform box with corner marks */}
    <rect
      x={KEY_BOX.x}
      y={KEY_BOX.y}
      width={KEY_BOX.w}
      height={KEY_BOX.h}
      className={styles.chromeDashRect}
      {...fine}
      strokeDasharray="2 3"
    />
    {[
      [KEY_BOX.x, KEY_BOX.y],
      [KEY_BOX.x + KEY_BOX.w, KEY_BOX.y],
      [KEY_BOX.x, KEY_BOX.y + KEY_BOX.h],
      [KEY_BOX.x + KEY_BOX.w, KEY_BOX.y + KEY_BOX.h],
    ].map(([cx, cy]) => (
      <rect
        key={`${cx}-${cy}`}
        x={cx - 1.5}
        y={cy - 1.5}
        width={3}
        height={3}
        className={styles.chromeFill}
      />
    ))}
    {/* Key box → events strip connector (events fly free from here) */}
    <line
      x1={KEY_BOX.x + KEY_BOX.w + 2}
      y1={AXIS}
      x2={216}
      y2={AXIS}
      className={styles.chromeDash}
      {...fine}
    />
    {/* Events strip: dashed frame, corner marks, one slot tick per row */}
    <rect
      x={STRIP.x}
      y={STRIP.y}
      width={STRIP.w}
      height={STRIP.h}
      className={styles.chromeDashRect}
      {...fine}
      strokeDasharray="2 3"
    />
    {[
      [STRIP.x, STRIP.y],
      [STRIP.x + STRIP.w, STRIP.y],
      [STRIP.x, STRIP.y + STRIP.h],
      [STRIP.x + STRIP.w, STRIP.y + STRIP.h],
    ].map(([cx, cy]) => (
      <rect
        key={`${cx}-${cy}`}
        x={cx - 1.5}
        y={cy - 1.5}
        width={3}
        height={3}
        className={styles.chromeFill}
      />
    ))}
    {ROW_Y.map((ry) => (
      <line key={ry} x1={246} y1={ry} x2={258} y2={ry} className={styles.chromeTick} {...fine} />
    ))}
  </svg>
);

const StageLabel = ({ x, y, children }: { x: number; y: number; children: ReactNode }) => (
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

// ─── Export ────────────────────────────────────────────────────────────────

const ARIA_LABEL =
  "Animated diagram of Hatchet’s built-in webhook ingestors: Stripe, GitHub, and Slack post webhook requests that converge on a Hatchet endpoint. Each request pauses at a signature-validation gate — one forged request flickers and is dropped — then a CEL event-key expression derives a routable key from the payload, like stripe:invoice.paid. The keyed event lands in an events strip and fires a run indicator for the subscribed workflow.";

export const WebhookIngest = ({ style }: { style?: CSSProperties }) => (
  <div className={styles.wrap} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.root}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={34} y={LABEL_Y}>
          Sources
        </StageLabel>
        <StageLabel x={GATE_X} y={LABEL_Y}>
          Validate
        </StageLabel>
        <StageLabel x={KEY_X} y={LABEL_Y}>
          Event key
        </StageLabel>
        <StageLabel x={STRIP.x + STRIP.w / 2} y={LABEL_Y}>
          Events
        </StageLabel>
        {SOURCES.map((label, i) => (
          <div
            key={label}
            className={styles.chip}
            style={{
              left: `calc(var(--flow-u) * ${CHIP_RIGHT})`,
              top: `calc(var(--flow-u) * ${CHIP_Y[i]})`,
            }}
          >
            {label}
          </div>
        ))}
        {RUN_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.runGlyph} />
          </Flow.Token>
        ))}
        {READOUT_TRACKS.map((track, i) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.readout}>
              <div className={styles.readoutExpr}>{VALID_HOOKS[i].expr}</div>
              <div className={styles.readoutSwap}>
                <span className={styles.readoutPayload}>{VALID_HOOKS[i].payload}</span>
                <span className={styles.readoutKey}>{VALID_HOOKS[i].key}</span>
              </div>
            </div>
          </Flow.Token>
        ))}
        <Flow.Token track={NOTE_TRACK}>
          <div className={styles.note}>invalid signature</div>
        </Flow.Token>
        {HOOK_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.dot} />
          </Flow.Token>
        ))}
      </Flow.Stage>
    </Flow.Root>
    <Text.Small as="p" secondary balance className={`${styles.caption}`}>
      Webhooks create Hatchet events after validation, which can trigger or resume any downstream
      workflow.
    </Text.Small>
  </div>
);
