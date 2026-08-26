"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  createSeededRandom,
  defineTrack,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import styles from "./streamingdemo.module.css";

/**
 * Flow animation: response streaming (docs.hatchet.run/v1/streaming). A user
 * message lands in a frontend chat panel and a request travels back to an
 * agent task running in a worker. While the task runs, it emits a steady
 * stream of chunks (`put_stream`) that flow along a dashed stream line
 * through a Hatchet relay node into the chat panel, where the response text
 * builds up word by word — streaming, not waiting for completion. The task
 * completes green while the final chunks are still in flight; the last chunk
 * lands, the caret settles, and after a brief dwell everything fades for a
 * seamless wrap.
 *
 * All tracks are plain data at module scope (seeded, deterministic): server
 * markup, the reduced-motion poster (mid-stream: partial response + running
 * task), and every loop iteration are reproducible.
 */

// ─── Geometry (stage design units, 320 × 220 — half-column composition) ────

const STAGE_W = 320;
const STAGE_H = 220;

const WORKER = { x: 14, y: 76, w: 76, h: 68 };
const AGENT = { x: 52, y: 110 };
const STREAM_Y = 110;
const RELAY = { x: 143, y: STREAM_Y };
const PANEL = { x: 198, y: 30, w: 112, h: 160 };
const HEADER_SEP_Y = 52;
const DOT = { x: 207, y: 41 };
const BUBBLE = { x: 270, y: 71 };

/** Response text region inside the panel. */
const INNER_X0 = 207;
const INNER_X1 = 301;
const ROW_Y0 = 101;
const ROW_PITCH = 12;
const ROWS = 6;
const WORD_GAP = 4;

// ─── Timing (ms) ───────────────────────────────────────────────────────────

const USER_AT = 350; // user message appears in the chat
const REQ_SPAWN = 750; // request leaves the frontend
const AGENT_RUN_AT = 1750; // agent task starts processing
const CARET_AT = 1900; // caret starts blinking at the response line
const EMIT0 = 2350; // first chunk emitted
const TRAVEL = 950; // chunk flight time, agent → panel

// ─── Response layout + stream schedule (seeded, deterministic) ─────────────

const rng = createSeededRandom("response-streaming");

interface WordBar {
  cx: number;
  cy: number;
  w: number;
}

/** Word-bar layout: rows fill left → right; the last row stays partial. */
const BARS: WordBar[] = [];
{
  let row = 0;
  let x = INNER_X0;
  let inRow = 0;
  while (row < ROWS) {
    const w = Math.round(12 + rng() * 18);
    if (x + w > INNER_X1) {
      row += 1;
      x = INNER_X0;
      inRow = 0;
      continue;
    }
    if (row === ROWS - 1 && inRow >= 2) break;
    BARS.push({ cx: x + w / 2, cy: ROW_Y0 + row * ROW_PITCH, w });
    x += w + WORD_GAP;
    inRow += 1;
  }
}

/** One chunk per word bar, emitted at a bursty LLM-ish cadence. */
const EMITS: number[] = [];
{
  let t = EMIT0;
  for (let i = 0; i < BARS.length; i++) {
    EMITS.push(Math.round(t));
    t += 290 + rng() * 150 + (rng() < 0.1 ? 400 : 0);
  }
}
const LANDS = EMITS.map((e) => e + TRAVEL);
const LAST_EMIT = EMITS[EMITS.length - 1];
const LAST_LAND = LANDS[LANDS.length - 1];

/** The task finishes emitting (flips green) while the tail chunks are still in flight. */
const DONE_AT = LAST_EMIT + 300;
const FADE_AT = LAST_LAND + 1300;
const DURATION = Math.ceil((FADE_AT + 700) / 100) * 100;
/** Static frame: ~55% of the response built, chunks mid-flight, task running. */
const POSTER_TIME = EMITS[Math.floor(BARS.length * 0.55)] + 450;

// ─── Tracks ────────────────────────────────────────────────────────────────

/** Chunk flight time to reach stage x (constant speed after the fade-in). */
const flightT = (e: number, x: number) => Math.round(e + 150 + ((x - 70) / 127) * (TRAVEL - 150));

const CHUNK_TRACKS: FlowTrack[] = BARS.map((_, i) => {
  const e = EMITS[i];
  return defineTrack(`chunk-${i}`, [
    { t: e, x: 58, y: STREAM_Y, opacity: 0, state: "chunk" },
    { t: e + 150, x: 70, y: STREAM_Y, opacity: 1, ease: "linear" },
    { t: flightT(e, RELAY.x), x: RELAY.x, y: STREAM_Y, ease: "linear" },
    { t: flightT(e, 181), x: 181, y: STREAM_Y, ease: "linear" },
    // Dissolves into the panel edge exactly as its word bar pops in.
    { t: e + TRAVEL, x: 197, y: STREAM_Y, opacity: 0, ease: "linear" },
  ]);
});

/** Word bars land accent-bright, then settle into text; all fade at the wrap. */
const BAR_TRACKS: FlowTrack[] = BARS.map((bar, i) =>
  defineTrack(`bar-${i}`, [
    { t: LANDS[i] - 40, x: bar.cx, y: bar.cy, opacity: 0, state: "fresh" },
    { t: LANDS[i], x: bar.cx, y: bar.cy, opacity: 1, ease: "linear" },
    { t: LANDS[i] + 550, x: bar.cx, y: bar.cy, ease: "hold", state: "settled" },
    { t: FADE_AT, x: bar.cx, y: bar.cy, ease: "hold" },
    { t: FADE_AT + 450, x: bar.cx, y: bar.cy, opacity: 0, ease: "linear" },
  ])
);

/** Blinking caret: waits at the response line, hops after each landed word. */
const CARET_TRACK = (() => {
  const kfs: FlowKeyframe[] = [
    { t: CARET_AT, x: INNER_X0 + 1, y: ROW_Y0, opacity: 0, state: "typing" },
    { t: CARET_AT + 250, x: INNER_X0 + 1, y: ROW_Y0, opacity: 1, ease: "linear" },
  ];
  BARS.forEach((bar, i) => {
    kfs.push({
      t: LANDS[i],
      x: Math.min(bar.cx + bar.w / 2 + 3, INNER_X1 + 1),
      y: bar.cy,
      ease: "hold",
      ...(i === BARS.length - 1 ? { state: "done" } : null),
    });
  });
  const last = kfs[kfs.length - 1];
  kfs.push(
    { t: LAST_LAND + 1000, x: last.x, y: last.y, ease: "hold" },
    { t: LAST_LAND + 1350, x: last.x, y: last.y, opacity: 0, ease: "linear" }
  );
  return defineTrack("caret", kfs);
})();

/** The user's prompt, heading back from the frontend to the agent. */
const REQUEST_TRACK = defineTrack("request", [
  { t: REQ_SPAWN, x: 197, y: STREAM_Y, opacity: 0, state: "request" },
  { t: REQ_SPAWN + 150, x: 188, y: STREAM_Y, opacity: 1, ease: "linear" },
  { t: 1320, x: RELAY.x, y: STREAM_Y, ease: "linear" },
  { t: 1720, x: 88, y: STREAM_Y, ease: "linear" },
  { t: 1850, x: 64, y: STREAM_Y, opacity: 0, ease: "linear" },
]);

/** The agent task pill: idle → running (accent) → done (green) → idle. */
const AGENT_TRACK = defineTrack("agent", [
  { t: 0, x: AGENT.x, y: AGENT.y, state: "idle" },
  { t: AGENT_RUN_AT, x: AGENT.x, y: AGENT.y, state: "running", ease: "hold" },
  { t: DONE_AT, x: AGENT.x, y: AGENT.y, state: "done", ease: "hold" },
  { t: FADE_AT + 250, x: AGENT.x, y: AGENT.y, state: "idle", ease: "hold" },
  { t: DURATION, x: AGENT.x, y: AGENT.y, ease: "hold" },
]);

/** The Hatchet relay node blips as each event passes through it. */
const RELAY_TRACK = (() => {
  const passes = [1320, ...EMITS.map((e) => flightT(e, RELAY.x))];
  const kfs: FlowKeyframe[] = [{ t: 0, x: RELAY.x, y: RELAY.y, state: "idle" }];
  for (const p of passes) {
    kfs.push(
      { t: p, x: RELAY.x, y: RELAY.y, state: "ping", ease: "hold" },
      { t: p + 260, x: RELAY.x, y: RELAY.y, state: "idle", ease: "hold" }
    );
  }
  kfs.push({ t: DURATION, x: RELAY.x, y: RELAY.y, ease: "hold" });
  return defineTrack("relay", kfs);
})();

/** Chat header status dot: idle → busy while streaming → green on completion. */
const DOT_TRACK = defineTrack("panel-dot", [
  { t: 0, x: DOT.x, y: DOT.y, state: "idle" },
  { t: REQ_SPAWN, x: DOT.x, y: DOT.y, state: "busy", ease: "hold" },
  { t: LAST_LAND, x: DOT.x, y: DOT.y, state: "ok", ease: "hold" },
  { t: FADE_AT + 250, x: DOT.x, y: DOT.y, state: "idle", ease: "hold" },
  { t: DURATION, x: DOT.x, y: DOT.y, ease: "hold" },
]);

/** The user's message bubble in the chat. */
const BUBBLE_TRACK = defineTrack("bubble", [
  { t: USER_AT, x: BUBBLE.x, y: BUBBLE.y, opacity: 0 },
  { t: USER_AT + 280, x: BUBBLE.x, y: BUBBLE.y, opacity: 1, ease: "linear" },
  { t: FADE_AT, x: BUBBLE.x, y: BUBBLE.y, ease: "hold" },
  { t: FADE_AT + 450, x: BUBBLE.x, y: BUBBLE.y, opacity: 0, ease: "linear" },
]);

// ─── Static chrome ─────────────────────────────────────────────────────────

const fine = { fill: "none", strokeWidth: 1, vectorEffect: "non-scaling-stroke" } as const;
const stroke = { fill: "none", strokeWidth: 1.5, vectorEffect: "non-scaling-stroke" } as const;

const Chrome = () => (
  <svg viewBox={`0 0 ${STAGE_W} ${STAGE_H}`} aria-hidden="true">
    {/* Worker box */}
    <rect
      x={WORKER.x}
      y={WORKER.y}
      width={WORKER.w}
      height={WORKER.h}
      className={styles.chromeStroke}
      {...stroke}
    />
    {/* Stream line: ports at both ends, dashed, broken around the relay node */}
    <rect x={94} y={STREAM_Y - 2} width={4} height={4} className={styles.chromeFill} />
    <line x1={102} y1={STREAM_Y} x2={134} y2={STREAM_Y} className={styles.chromeDash} {...fine} />
    <line x1={152} y1={STREAM_Y} x2={186} y2={STREAM_Y} className={styles.chromeDash} {...fine} />
    <rect x={190} y={STREAM_Y - 2} width={4} height={4} className={styles.chromeFill} />
    {/* Frontend chat panel (dashboard-mock chrome) */}
    <rect
      x={PANEL.x}
      y={PANEL.y}
      width={PANEL.w}
      height={PANEL.h}
      rx={4}
      className={styles.chromePanel}
      strokeWidth={1}
      vectorEffect="non-scaling-stroke"
    />
    <line
      x1={PANEL.x}
      y1={HEADER_SEP_Y}
      x2={PANEL.x + PANEL.w}
      y2={HEADER_SEP_Y}
      className={styles.chromeSep}
      {...fine}
    />
    {/* Chat input mock at the panel's bottom */}
    <rect
      x={206}
      y={168}
      width={96}
      height={14}
      rx={3}
      className={styles.chromeInput}
      strokeWidth={1}
      vectorEffect="non-scaling-stroke"
    />
    <rect x={293} y={172} width={6} height={6} className={styles.chromeInputSend} />
  </svg>
);

const StageLabel = ({
  x,
  y,
  anchor = "center",
  muted = false,
  code = false,
  children,
}: {
  x: number;
  y: number;
  anchor?: "center" | "left";
  muted?: boolean;
  code?: boolean;
  children: ReactNode;
}) => (
  <div
    className={`${styles.stageLabel} ${anchor === "left" ? styles.anchorLeft : ""} ${
      muted ? styles.labelMuted : ""
    } ${code ? styles.labelCode : ""}`}
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
  "Animated diagram of Hatchet response streaming: a user message in a frontend chat panel sends a request to an agent task running in a worker; while the task is still running it emits a stream of chunks that flow through a Hatchet relay node into the chat, where the response builds up progressively, until the task completes and the final chunk lands.";

export const StreamingDemo = ({ style }: { style?: CSSProperties }) => (
  <div className={styles.wrap} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.root}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={WORKER.x + WORKER.w / 2} y={64}>
          worker
        </StageLabel>
        <StageLabel x={RELAY.x} y={124}>
          hatchet
        </StageLabel>
        <StageLabel x={214} y={38} anchor="left">
          frontend
        </StageLabel>
        <StageLabel x={AGENT.x} y={126} muted code>
          put_stream()
        </StageLabel>
        <Flow.Token track={BUBBLE_TRACK}>
          <div className={styles.userBubble}>
            <div className={styles.userLine} style={{ width: "calc(var(--flow-u) * 46)" }} />
            <div className={styles.userLine} style={{ width: "calc(var(--flow-u) * 27)" }} />
          </div>
        </Flow.Token>
        {BAR_TRACKS.map((track, i) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.wordBar} style={{ width: `calc(var(--flow-u) * ${BARS[i].w})` }} />
          </Flow.Token>
        ))}
        <Flow.Token track={CARET_TRACK}>
          <div className={styles.caret} />
        </Flow.Token>
        {CHUNK_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.chunk} />
          </Flow.Token>
        ))}
        <Flow.Token track={REQUEST_TRACK}>
          <div className={styles.req} />
        </Flow.Token>
        {/* Relay renders above the stream tokens, so they pass through it */}
        <Flow.Token track={RELAY_TRACK}>
          <div className={styles.relay} />
        </Flow.Token>
        <Flow.Token track={AGENT_TRACK}>
          <div className={styles.agentPill}>agent</div>
        </Flow.Token>
        <Flow.Token track={DOT_TRACK}>
          <div className={styles.panelDot} />
        </Flow.Token>
      </Flow.Stage>
    </Flow.Root>
  </div>
);
