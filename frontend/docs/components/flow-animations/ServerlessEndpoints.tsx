"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  defineTrack,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import styles from "./serverlessendpoints.module.css";

const STAGE_W = 320;
const STAGE_H = 220;

const ENGINE = { x: 12, y: 70, w: 66, h: 80 };
const OPERATOR = { x: 130, y: 84, w: 60, h: 52 };
const ENDPOINT = { x: 242, y: 70, w: 66, h: 80 };
const UP = 98;
const DOWN = 124;

const ENGINE_EDGE = ENGINE.x + ENGINE.w + 2;
const OP_LEFT = OPERATOR.x - 2;
const OP_RIGHT = OPERATOR.x + OPERATOR.w + 2;
const ENDPOINT_EDGE = ENDPOINT.x - 2;
const LEFT_MID = (ENGINE_EDGE + OP_LEFT) / 2;
const RIGHT_MID = (OP_RIGHT + ENDPOINT_EDGE) / 2;

const SLOT = 7;
const slotX = (i: number) => ENGINE.x + 21 + i * 12;
const TASK = { x: ENGINE.x + ENGINE.w / 2, y: DOWN };
const RUN = { x: ENDPOINT.x + ENDPOINT.w / 2, y: DOWN + 4 };
const endpointCenter = {
  x: ENDPOINT.x + ENDPOINT.w / 2,
  y: ENDPOINT.y + ENDPOINT.h / 2,
};

const SPEED = 0.065;

type Pt = [number, number];

const along = (a: Pt, b: Pt, f: number): Pt => [
  a[0] + (b[0] - a[0]) * f,
  a[1] + (b[1] - a[1]) * f,
];

const hop = (id: string, t0: number, a: Pt, b: Pt, state: string) => {
  const duration = Math.round(Math.hypot(b[0] - a[0], b[1] - a[1]) / SPEED);
  const frame = (f: number, opacity: number): FlowKeyframe => {
    const [x, y] = along(a, b, f);
    return {
      t: Math.round(t0 + duration * f),
      x,
      y,
      opacity,
      ease: "linear",
    };
  };
  return {
    start: t0,
    end: t0 + duration,
    track: defineTrack(id, [
      { ...frame(0, 0), state },
      frame(0.12, 1),
      frame(0.88, 1),
      frame(1, 0),
    ]),
  };
};

const POLL = hop("poll-1", 700, [OP_RIGHT, UP], [ENDPOINT_EDGE, UP], "poll");
const CATALOG = hop(
  "catalog",
  POLL.end + 250,
  [ENDPOINT_EDGE, UP],
  [OP_RIGHT, UP],
  "poll",
);
const REGISTER = hop(
  "register",
  CATALOG.end + 200,
  [OP_LEFT, UP],
  [ENGINE_EDGE, UP],
  "poll",
);

const QUEUED_AT = REGISTER.end + 900;
const ASSIGN = hop(
  "assign",
  QUEUED_AT + 500,
  [ENGINE_EDGE, DOWN],
  [OP_LEFT, DOWN],
  "task",
);
const TRIGGER = hop(
  "trigger",
  ASSIGN.end + 150,
  [OP_RIGHT, DOWN],
  [ENDPOINT_EDGE, DOWN],
  "task",
);
const RUN_END = TRIGGER.end + 1100;
const OUTPUT = hop(
  "output",
  RUN_END,
  [ENDPOINT_EDGE, DOWN],
  [OP_RIGHT, DOWN],
  "ok",
);
const COMPLETE = hop(
  "complete",
  OUTPUT.end + 150,
  [OP_LEFT, DOWN],
  [ENGINE_EDGE, DOWN],
  "ok",
);

const POLL_AGAIN = hop(
  "poll-2",
  COMPLETE.end + 600,
  [OP_RIGHT, UP],
  [ENDPOINT_EDGE, UP],
  "poll",
);
const CATALOG_AGAIN = hop(
  "catalog-2",
  POLL_AGAIN.end + 250,
  [ENDPOINT_EDGE, UP],
  [OP_RIGHT, UP],
  "poll",
);

const HOPS = [
  POLL,
  CATALOG,
  REGISTER,
  ASSIGN,
  TRIGGER,
  OUTPUT,
  COMPLETE,
  POLL_AGAIN,
  CATALOG_AGAIN,
];

const FADE_AT = CATALOG_AGAIN.end + 900;
const DURATION = Math.ceil((FADE_AT + 800) / 100) * 100;
const POSTER_TIME = Math.round((TRIGGER.start + TRIGGER.end) / 2);

const fadeOut = (x: number, y: number): FlowKeyframe[] => [
  { t: FADE_AT, x, y, ease: "hold" },
  { t: FADE_AT + 450, x, y, opacity: 0, ease: "linear" },
];

type Caption = { text: string; x: number; y: number; from: number; to: number };

const CAPTION_UP = UP - 13;
const CAPTION_DOWN = DOWN + 7;
const CAPTIONS: Caption[] = [
  {
    text: "healthcheck",
    x: RIGHT_MID,
    y: CAPTION_UP,
    from: POLL.start,
    to: POLL.end,
  },
  {
    text: "workflows",
    x: RIGHT_MID,
    y: CAPTION_UP,
    from: CATALOG.start,
    to: CATALOG.end,
  },
  {
    text: "register",
    x: LEFT_MID,
    y: CAPTION_UP,
    from: REGISTER.start,
    to: REGISTER.end,
  },
  {
    text: "assign",
    x: LEFT_MID,
    y: CAPTION_DOWN,
    from: ASSIGN.start,
    to: ASSIGN.end,
  },
  {
    text: "trigger",
    x: RIGHT_MID,
    y: CAPTION_DOWN,
    from: TRIGGER.start,
    to: TRIGGER.end,
  },
  {
    text: "output",
    x: RIGHT_MID,
    y: CAPTION_DOWN,
    from: OUTPUT.start,
    to: OUTPUT.end,
  },
  {
    text: "completed",
    x: LEFT_MID,
    y: CAPTION_DOWN,
    from: COMPLETE.start,
    to: COMPLETE.end,
  },
  {
    text: "healthcheck",
    x: RIGHT_MID,
    y: CAPTION_UP,
    from: POLL_AGAIN.start,
    to: POLL_AGAIN.end,
  },
  {
    text: "workflows",
    x: RIGHT_MID,
    y: CAPTION_UP,
    from: CATALOG_AGAIN.start,
    to: CATALOG_AGAIN.end,
  },
];

const CAPTION_TRACKS: FlowTrack[] = CAPTIONS.map((c, i) =>
  defineTrack(`caption-${i}`, [
    { t: c.from - 120, x: c.x, y: c.y, opacity: 0 },
    { t: c.from, x: c.x, y: c.y, opacity: 1, ease: "linear" },
    { t: c.to, x: c.x, y: c.y, ease: "hold" },
    { t: c.to + 120, x: c.x, y: c.y, opacity: 0, ease: "linear" },
  ]),
);

const SLOT_TRACKS: FlowTrack[] = [0, 1, 2].map((i) => {
  const x = slotX(i);
  const at = REGISTER.end + i * 140;
  return defineTrack(`slot-${i}`, [
    { t: at - 40, x, y: UP, opacity: 0, state: "fresh" },
    { t: at, x, y: UP, opacity: 1, ease: "linear" },
    { t: at + 600, x, y: UP, state: "stored", ease: "hold" },
    ...fadeOut(x, UP),
  ]);
});

const TASK_TRACK = defineTrack("task", [
  { t: QUEUED_AT - 200, x: TASK.x, y: TASK.y, opacity: 0, state: "queued" },
  { t: QUEUED_AT, x: TASK.x, y: TASK.y, opacity: 1, ease: "linear" },
  { t: ASSIGN.start, x: TASK.x, y: TASK.y, state: "running", ease: "hold" },
  { t: COMPLETE.end, x: TASK.x, y: TASK.y, state: "done", ease: "hold" },
  ...fadeOut(TASK.x, TASK.y),
]);

const RUN_TRACK = defineTrack("run", [
  { t: TRIGGER.end - 40, x: RUN.x, y: RUN.y, opacity: 0, state: "running" },
  { t: TRIGGER.end, x: RUN.x, y: RUN.y, opacity: 1, ease: "linear" },
  { t: RUN_END, x: RUN.x, y: RUN.y, state: "done", ease: "hold" },
  { t: RUN_END + 500, x: RUN.x, y: RUN.y, ease: "hold" },
  { t: RUN_END + 800, x: RUN.x, y: RUN.y, opacity: 0, ease: "linear" },
]);

const ENDPOINT_TRACK = defineTrack("endpoint", [
  { ...endpointCenter, t: 0, state: "waiting" },
  { ...endpointCenter, t: POLL.end, state: "healthy", ease: "hold" },
  { ...endpointCenter, t: TRIGGER.end, state: "busy", ease: "hold" },
  { ...endpointCenter, t: RUN_END, state: "healthy", ease: "hold" },
  { ...endpointCenter, t: FADE_AT + 250, state: "waiting", ease: "hold" },
  { ...endpointCenter, t: DURATION, ease: "hold" },
]);

const fine = {
  fill: "none",
  strokeWidth: 1,
  vectorEffect: "non-scaling-stroke",
} as const;
const stroke = {
  fill: "none",
  strokeWidth: 1.5,
  vectorEffect: "non-scaling-stroke",
} as const;

const Chrome = () => (
  <svg viewBox={`0 0 ${STAGE_W} ${STAGE_H}`} aria-hidden="true">
    {[ENGINE, OPERATOR].map((box) => (
      <rect
        key={box.x}
        x={box.x}
        y={box.y}
        width={box.w}
        height={box.h}
        className={styles.chromeStroke}
        {...stroke}
      />
    ))}
    {[0, 1, 2].map((i) => (
      <rect
        key={i}
        x={slotX(i) - SLOT / 2}
        y={UP - SLOT / 2}
        width={SLOT}
        height={SLOT}
        className={styles.chromeSlot}
        {...fine}
      />
    ))}
    {[UP, DOWN].map((y) => (
      <g key={y}>
        <line
          x1={ENGINE_EDGE}
          y1={y}
          x2={OP_LEFT}
          y2={y}
          className={styles.chromeDash}
          {...fine}
        />
        <line
          x1={OP_RIGHT}
          y1={y}
          x2={ENDPOINT_EDGE}
          y2={y}
          className={styles.chromeDash}
          {...fine}
        />
      </g>
    ))}
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

const ARIA_LABEL =
  "Animated diagram of a Hatchet serverless endpoint. The serverless operator sends a signed healthcheck to your endpoint, the endpoint answers with the workflows it serves, and the operator registers them with the Hatchet engine. When the engine assigns a task, the operator posts a signed trigger request to the endpoint, the endpoint runs the task and returns its output, and the operator reports the task as completed to the engine. The healthcheck then repeats on the next poll interval.";

export const ServerlessEndpoints = ({ style }: { style?: CSSProperties }) => (
  <div className={styles.wrap} style={style}>
    <Flow.Root
      duration={DURATION}
      posterTime={POSTER_TIME}
      className={styles.root}
      aria-label={ARIA_LABEL}
    >
      <Flow.Stage width={STAGE_W} height={STAGE_H} className={styles.stage}>
        <Chrome />
        <StageLabel x={ENGINE.x + ENGINE.w / 2} y={ENGINE.y - 12}>
          hatchet engine
        </StageLabel>
        <StageLabel x={OPERATOR.x + OPERATOR.w / 2} y={OPERATOR.y - 22} muted>
          serverless
        </StageLabel>
        <StageLabel x={OPERATOR.x + OPERATOR.w / 2} y={OPERATOR.y - 12}>
          operator
        </StageLabel>
        <StageLabel x={ENGINE.x + ENGINE.w / 2} y={UP - 16} muted>
          workflows
        </StageLabel>
        <StageLabel x={OPERATOR.x + OPERATOR.w / 2} y={STAGE_H - 46} muted>
          every poll interval
        </StageLabel>
        <Flow.Token track={ENDPOINT_TRACK}>
          <div className={styles.endpoint}>
            <div className={styles.endpointHeader}>
              <span className={styles.endpointDot} />
              your endpoint
            </div>
          </div>
        </Flow.Token>
        {SLOT_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.slot} />
          </Flow.Token>
        ))}
        <Flow.Token track={RUN_TRACK}>
          <div className={styles.run}>fn()</div>
        </Flow.Token>
        {HOPS.map((h) => (
          <Flow.Token key={h.track.id} track={h.track}>
            <div className={styles.msg} />
          </Flow.Token>
        ))}
        {CAPTION_TRACKS.map((track, i) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.caption}>{CAPTIONS[i].text}</div>
          </Flow.Token>
        ))}
        <Flow.Token track={TASK_TRACK}>
          <div className={styles.taskPill}>task</div>
        </Flow.Token>
      </Flow.Stage>
    </Flow.Root>
  </div>
);
