"use client";

import type { CSSProperties } from "react";
import { Flow, defineTrack, type FlowKeyframe, type FlowTrack } from "@/components/flow";
import { Text } from "@/components/flow/Text";
import styles from "./embeddedmode.module.css";

/**
 * Flow animation for embedded mode, told as a merge. The loop opens on the
 * standard deployment: three separate services — worker, engine, postgres —
 * each in its own dashed process boundary, chattering over dashed network
 * links (dispatch to the worker, result back, write to the database). Then
 * the links fall away and the three boxes glide together into a single dashed
 * boundary labelled "worker process": the same components, the same chatter,
 * now in-process hops with nothing else to run.
 *
 * Unlike the steady-state animations, this is a narrative loop: it ends with
 * a short crossfade back to the separated layout, so t = DURATION matches
 * t = 0 exactly. Every element that moves or fades is a token; the static
 * SVG chrome is empty.
 */

// ─── Geometry (stage design units, 320 × 208 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 208;

const COMPONENTS = ["worker", "engine", "postgres"] as const;
type Component = (typeof COMPONENTS)[number];

/** Phase 1 — three separate services in a row in the upper half. */
const SEP_Y = 86;
const SEP_X: Record<Component, number> = { worker: 52, engine: 160, postgres: 268 };

/** Phase 2 — the same boxes inside one process boundary at center. */
const PROC_Y = 128;
const PROC_X: Record<Component, number> = { worker: 86, engine: 160, postgres: 234 };
const PROC_BOUND = { cx: 160, cy: PROC_Y };
const PROC_LABEL_Y = 84;

/** Network links between the separated services (midpoint + span). */
const LINKS = [
  { id: "we", cx: 106, span: 60 },
  { id: "ep", cx: 214, span: 60 },
] as const;
/** In-process connectors between the merged boxes. */
const CONNECTORS = [
  { id: "we", cx: 123, span: 30 },
  { id: "ep", cx: 197, span: 30 },
] as const;
const NETWORK_LABEL_Y = SEP_Y + 8;

// ─── Timing (ms) ───────────────────────────────────────────────────────────

const DURATION = 16000;

/** Phase boundaries. */
const MERGE_FADE = 4500; // links + per-service boundaries start to go
const MERGE_START = 5000; // boxes begin to travel
const MERGE_END = 6200; // boxes in place
const ASSEMBLED = 6700; // process boundary fully in
const TEARDOWN = 14800; // crossfade back to the separated layout begins

const FADE = 400;
const DOT_FADE = 120;

/**
 * Static frame: mid-way through the merged phase — the three boxes sit inside
 * the "worker process" boundary and a chatter dot is crossing from the engine
 * to postgres. The punchline is legible without motion.
 */
const POSTER_TIME = 8450;

// ─── Component boxes (persistent tokens: separate → merged → reset) ────────

/**
 * Each box lives for the whole loop: it opens separated, glides into the
 * process, and crossfades back for the wrap.
 */
const boxTrack = (c: Component): FlowTrack =>
  defineTrack(`em-box-${c}`, [
    { t: 0, x: SEP_X[c], y: SEP_Y },
    { t: MERGE_START, x: SEP_X[c], y: SEP_Y, ease: "hold" },
    { t: MERGE_END, x: PROC_X[c], y: PROC_Y, ease: "inOut" },
    { t: TEARDOWN + 100, x: PROC_X[c], y: PROC_Y, ease: "hold" },
    { t: TEARDOWN + 500, x: PROC_X[c], y: PROC_Y, opacity: 0, ease: "linear" },
    { t: TEARDOWN + 600, x: SEP_X[c], y: SEP_Y, ease: "hold" },
    { t: DURATION, x: SEP_X[c], y: SEP_Y, opacity: 1, ease: "linear" },
  ]);

const BOX_TRACKS = COMPONENTS.map((c) => ({ component: c, track: boxTrack(c) }));

// ─── Phase scenery (links, labels, boundary — fading tokens) ───────────────

/** Visible through phase 1, gone for the merged phase, back for the wrap. */
const separatedSceneTrack = (id: string, x: number, y: number): FlowTrack =>
  defineTrack(id, [
    { t: 0, x, y, opacity: 1 },
    { t: MERGE_FADE, x, y, ease: "hold" },
    { t: MERGE_FADE + FADE, x, y, opacity: 0, ease: "linear" },
    { t: TEARDOWN + 700, x, y, ease: "hold" },
    { t: DURATION, x, y, opacity: 1, ease: "linear" },
  ]);

/** Hidden through phase 1, in for the merged phase, gone again for the wrap. */
const mergedSceneTrack = (id: string, x: number, y: number): FlowTrack =>
  defineTrack(id, [
    { t: 0, x, y, opacity: 0 },
    { t: MERGE_END, x, y, ease: "hold" },
    { t: ASSEMBLED, x, y, opacity: 1, ease: "linear" },
    { t: TEARDOWN, x, y, ease: "hold" },
    { t: TEARDOWN + FADE, x, y, opacity: 0, ease: "linear" },
    { t: DURATION, x, y, opacity: 0, ease: "hold" },
  ]);

const LINK_TRACKS = LINKS.map((l) => ({
  span: l.span,
  track: separatedSceneTrack(`em-link-${l.id}`, l.cx, SEP_Y),
}));

const NETWORK_LABEL_TRACKS = LINKS.map((l) => ({
  track: separatedSceneTrack(`em-netlabel-${l.id}`, l.cx, NETWORK_LABEL_Y),
}));

const CONNECTOR_TRACKS = CONNECTORS.map((c) => ({
  span: c.span,
  track: mergedSceneTrack(`em-conn-${c.id}`, c.cx, PROC_Y),
}));

const PROC_BOUND_TRACK = mergedSceneTrack("em-procbound", PROC_BOUND.cx, PROC_BOUND.cy);
const PROC_LABEL_TRACK = mergedSceneTrack("em-proclabel", PROC_BOUND.cx, PROC_LABEL_Y);

// ─── Chatter (dispatch → result → persist, in both phases) ─────────────────

/** One dot: fade in at the start point, travel, absorbed at the end point. */
const dotKeyframes = (t: number, x0: number, x1: number, y: number, travel: number): FlowKeyframe[] => [
  { t, x: x0, y, opacity: 0 },
  { t: t + 100, x: x0 + (x1 > x0 ? 4 : -4), y, opacity: 1, ease: "linear" },
  { t: t + travel, x: x1, y, ease: "linear" },
  { t: t + travel + DOT_FADE, x: x1, y, opacity: 0, ease: "linear" },
];

interface HopSpec {
  /** engine → worker, worker → engine, engine → postgres. */
  hops: [number, number][];
  y: number;
  travel: number;
  /** Offsets of the three hops from the cycle start. */
  beats: [number, number, number];
}

/** Separated: long hops across the network gaps. */
const SEPARATED_HOPS: HopSpec = {
  hops: [
    [138, 74],
    [74, 138],
    [182, 246],
  ],
  y: SEP_Y,
  travel: 500,
  beats: [0, 900, 1700],
};

/** Merged: the same conversation as short in-process hops. */
const MERGED_HOPS: HopSpec = {
  hops: [
    [140, 106],
    [106, 140],
    [180, 214],
  ],
  y: PROC_Y,
  travel: 300,
  beats: [0, 700, 1300],
};

const chatterCycle = (id: string, start: number, spec: HopSpec): FlowTrack[] =>
  spec.hops.map(([x0, x1], i) =>
    defineTrack(`${id}-${i}`, dotKeyframes(start + spec.beats[i], x0, x1, spec.y, spec.travel))
  );

const DOT_TRACKS: FlowTrack[] = [
  ...[600, 2500].flatMap((t, i) => chatterCycle(`em-sep-${i}`, t, SEPARATED_HOPS)),
  ...[7000, 8900, 10800, 12700].flatMap((t, i) => chatterCycle(`em-proc-${i}`, t, MERGED_HOPS)),
];

// ─── Export ────────────────────────────────────────────────────────────────

const ARIA_LABEL =
  "Animated diagram of Hatchet embedded mode. It opens on three separate services — a worker, the Hatchet engine, and a postgres database — each in its own dashed process boundary, exchanging messages over dashed network links. The links then fall away and the three boxes glide together into a single dashed boundary labelled worker process, where the same messages continue as short in-process hops.";

const CAPTION = "The engine and its database move in-process: one worker process, nothing else to run.";

export const EmbeddedMode = ({
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
        {LINK_TRACKS.map(({ span, track }) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.linkLine} style={{ width: `calc(var(--flow-u) * ${span})` }} />
          </Flow.Token>
        ))}
        {NETWORK_LABEL_TRACKS.map(({ track }) => (
          <Flow.Token key={track.id} track={track}>
            <div className={`${styles.tokenLabel} ${styles.labelMuted}`}>network</div>
          </Flow.Token>
        ))}
        {CONNECTOR_TRACKS.map(({ span, track }) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.linkLine} style={{ width: `calc(var(--flow-u) * ${span})` }} />
          </Flow.Token>
        ))}
        <Flow.Token track={PROC_BOUND_TRACK}>
          <div className={styles.processBoundary} />
        </Flow.Token>
        <Flow.Token track={PROC_LABEL_TRACK}>
          <div className={styles.tokenLabel}>worker process</div>
        </Flow.Token>
        {BOX_TRACKS.map(({ component, track }) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.component}>
              <div className={`${styles.componentBox} ${styles[component]}`} />
              <div className={`${styles.tokenLabel} ${styles.componentLabel}`}>{component}</div>
            </div>
          </Flow.Token>
        ))}
        {DOT_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.dot} />
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
