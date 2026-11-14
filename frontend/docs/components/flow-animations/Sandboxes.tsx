"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  defineTrack,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import styles from "./sandboxes.module.css";

const STAGE_W = 320;
const STAGE_H = 220;

const WORKER = { x: 14, y: 76, w: 76, h: 68 };
const TASK = { x: 52, y: 104 };
const CALL_LABEL = { x: 52, y: 122 };
const LINE_Y = 110;

const LOG = { x: 114, y: 100, w: 92, h: 20 };
const SLOT = 8;
const SLOT_PITCH = 14;
const slotX = (i: number) => LOG.x + 11 + i * SLOT_PITCH;

const JUNCTION_X = 220;
const PROVIDER = { x: 228, y: 28, w: 84, h: 166 };
const BOX = { w: 70, h: 44 };
const TOP = { x: 235, y: 38 };
const BOTTOM = { x: 235, y: 140 };
const SHELF = { x: 270, y: 111 };
const center = (b: { x: number; y: number }) => ({
  x: b.x + BOX.w / 2,
  y: b.y + BOX.h / 2,
});
const barX = (i: number) => TOP.x + 9 + i * 8;
const BAR_DY = 31;

const SPEED = 0.13;
const RUN_AT = 700;

type Pt = [number, number];
type Call = {
  label: string;
  at: number;
  target: typeof TOP;
  work: number;
};

const CALLS: Call[] = [
  { label: "create()", at: 1100, target: TOP, work: 500 },
  { label: "exec()", at: 2600, target: TOP, work: 700 },
  { label: "exec()", at: 3900, target: TOP, work: 700 },
  { label: "kill()", at: 5200, target: TOP, work: 900 },
  { label: "create({ from })", at: 7200, target: BOTTOM, work: 800 },
  { label: "connect()", at: 9000, target: TOP, work: 800 },
];
const CREATE = 0;
const EXECS = [1, 2];
const KILL = 3;
const FORK = 4;
const CONNECT = 5;

const pathLength = (pts: Pt[]) =>
  pts
    .slice(1)
    .reduce(
      (sum, [x, y], i) => sum + Math.hypot(x - pts[i][0], y - pts[i][1]),
      0,
    );

const alongPath = (t0: number, pts: Pt[], state: string): FlowKeyframe[] => {
  let travelled = 0;
  return pts.map(([x, y], i) => {
    if (i > 0) {
      travelled += Math.hypot(x - pts[i - 1][0], y - pts[i - 1][1]);
    }
    const kf: FlowKeyframe = {
      t: Math.round(t0 + travelled / SPEED),
      x,
      y,
      opacity: i === 0 || i === pts.length - 1 ? 0 : 1,
      ease: "linear",
    };
    return i === 0 ? { ...kf, state } : kf;
  });
};

const arrival = (t0: number, pts: Pt[]) =>
  Math.round(t0 + pathLength(pts) / SPEED);

const inPath = (i: number): Pt[] => [
  [60, LINE_Y],
  [72, LINE_Y],
  [slotX(i), LINE_Y],
];

const outPath = (target: { x: number; y: number }): Pt[] => {
  const y = center(target).y;
  return [
    [204, LINE_Y],
    [210, LINE_Y],
    [JUNCTION_X, LINE_Y],
    [JUNCTION_X, y],
    [target.x - 2, y],
  ];
};

const QUEUED = CALLS.map((c, i) => arrival(c.at, inPath(i)));
const DEPART = QUEUED.map((q) => q + 120);
const LAND = DEPART.map((d, i) => arrival(d, outPath(CALLS[i].target)));
const DONE = LAND.map((l, i) => l + CALLS[i].work);

const SNAP_DROP = LAND[KILL] + 150;
const SNAP_STORED = SNAP_DROP + 500;
const FORK_COPY = LAND[FORK] + 100;
const FORK_UP = FORK_COPY + 450;
const CONNECT_COPY = LAND[CONNECT] + 100;
const CONNECT_UP = CONNECT_COPY + 450;

const FADE_AT = DONE[CONNECT] + 1500;
const DURATION = Math.ceil((FADE_AT + 800) / 100) * 100;
const POSTER_TIME = FORK_COPY + 250;

const fadeOut = (x: number, y: number): FlowKeyframe[] => [
  { t: FADE_AT, x, y, ease: "hold" },
  { t: FADE_AT + 450, x, y, opacity: 0, ease: "linear" },
];

const IN_TRACKS: FlowTrack[] = CALLS.map((c, i) =>
  defineTrack(`in-${i}`, alongPath(c.at, inPath(i), "call")),
);

const OUT_TRACKS: FlowTrack[] = CALLS.map((c, i) =>
  defineTrack(`out-${i}`, alongPath(DEPART[i], outPath(c.target), "call")),
);

const SLOT_TRACKS: FlowTrack[] = CALLS.map((_, i) => {
  const x = slotX(i);
  return defineTrack(`slot-${i}`, [
    { t: QUEUED[i] - 40, x, y: LINE_Y, opacity: 0, state: "running" },
    { t: QUEUED[i], x, y: LINE_Y, opacity: 1, ease: "linear" },
    { t: DONE[i], x, y: LINE_Y, state: "done", ease: "hold" },
    ...fadeOut(x, LINE_Y),
  ]);
});

const LABEL_TRACKS: FlowTrack[] = CALLS.map((c, i) => {
  const end = i + 1 < CALLS.length ? CALLS[i + 1].at - 200 : DONE[i] + 300;
  const { x, y } = CALL_LABEL;
  return defineTrack(`label-${i}`, [
    { t: c.at - 200, x, y, opacity: 0 },
    { t: c.at, x, y, opacity: 1, ease: "linear" },
    { t: end - 150, x, y, ease: "hold" },
    { t: end, x, y, opacity: 0, ease: "linear" },
  ]);
});

const barTrack = (
  id: string,
  at: number,
  i: number,
  box: { y: number },
  state: string,
) => {
  const x = barX(i);
  const y = box.y + BAR_DY;
  return defineTrack(id, [
    { t: at - 40, x, y, opacity: 0, state },
    { t: at, x, y, opacity: 1, ease: "linear" },
    { t: at + 600, x, y, state: "settled", ease: "hold" },
    ...fadeOut(x, y),
  ]);
};

const BAR_TRACKS: FlowTrack[] = [
  ...EXECS.map((call, i) =>
    barTrack(`bar-top-${i}`, LAND[call], i, TOP, "fresh"),
  ),
  ...EXECS.map((_, i) =>
    barTrack(`bar-fork-${i}`, FORK_UP + i * 120, i, BOTTOM, "restored"),
  ),
];

const SNAPSHOT_TRACK = defineTrack("snapshot", [
  {
    t: SNAP_DROP,
    x: SHELF.x,
    y: TOP.y + BOX.h + 2,
    opacity: 0,
    state: "fresh",
  },
  {
    t: SNAP_DROP + 120,
    x: SHELF.x,
    y: TOP.y + BOX.h + 6,
    opacity: 1,
    ease: "linear",
  },
  { t: SNAP_STORED, x: SHELF.x, y: SHELF.y, ease: "out" },
  {
    t: SNAP_STORED + 500,
    x: SHELF.x,
    y: SHELF.y,
    state: "stored",
    ease: "hold",
  },
  { t: FORK_COPY, x: SHELF.x, y: SHELF.y, state: "read", ease: "hold" },
  { t: FORK_COPY + 350, x: SHELF.x, y: SHELF.y, state: "stored", ease: "hold" },
  { t: CONNECT_COPY, x: SHELF.x, y: SHELF.y, state: "read", ease: "hold" },
  {
    t: CONNECT_COPY + 350,
    x: SHELF.x,
    y: SHELF.y,
    state: "stored",
    ease: "hold",
  },
  ...fadeOut(SHELF.x, SHELF.y),
]);

const copyTrack = (id: string, t0: number, t1: number, toY: number) =>
  defineTrack(id, [
    { t: t0, x: SHELF.x, y: SHELF.y, opacity: 0, state: "copy" },
    { t: t0 + 80, x: SHELF.x, y: SHELF.y, opacity: 1, ease: "linear" },
    { t: t1 - 60, x: SHELF.x, y: toY, ease: "inOut" },
    { t: t1, x: SHELF.x, y: toY, opacity: 0, ease: "linear" },
  ]);

const COPY_TRACKS: FlowTrack[] = [
  copyTrack("copy-fork", FORK_COPY, FORK_UP, BOTTOM.y - 2),
  copyTrack("copy-connect", CONNECT_COPY, CONNECT_UP, TOP.y + BOX.h + 2),
];

const TASK_TRACK = defineTrack("task", [
  { t: 0, x: TASK.x, y: TASK.y, state: "idle" },
  { t: RUN_AT, x: TASK.x, y: TASK.y, state: "running", ease: "hold" },
  { t: DONE[CONNECT] + 300, x: TASK.x, y: TASK.y, state: "done", ease: "hold" },
  { t: FADE_AT + 250, x: TASK.x, y: TASK.y, state: "idle", ease: "hold" },
  { t: DURATION, x: TASK.x, y: TASK.y, ease: "hold" },
]);

const TOP_TRACK = defineTrack("sandbox-top", [
  { ...center(TOP), t: 0, state: "off" },
  { ...center(TOP), t: LAND[CREATE] + 300, state: "running", ease: "hold" },
  { ...center(TOP), t: SNAP_DROP, state: "stopped", ease: "hold" },
  { ...center(TOP), t: CONNECT_UP, state: "running", ease: "hold" },
  { ...center(TOP), t: FADE_AT + 250, state: "off", ease: "hold" },
  { ...center(TOP), t: DURATION, ease: "hold" },
]);

const BOTTOM_TRACK = defineTrack("sandbox-fork", [
  { ...center(BOTTOM), t: 0, state: "off" },
  { ...center(BOTTOM), t: FORK_UP, state: "running", ease: "hold" },
  { ...center(BOTTOM), t: FADE_AT + 250, state: "off", ease: "hold" },
  { ...center(BOTTOM), t: DURATION, ease: "hold" },
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
    <rect
      x={WORKER.x}
      y={WORKER.y}
      width={WORKER.w}
      height={WORKER.h}
      className={styles.chromeStroke}
      {...stroke}
    />
    <rect
      x={94}
      y={LINE_Y - 2}
      width={4}
      height={4}
      className={styles.chromeFill}
    />
    <line
      x1={102}
      y1={LINE_Y}
      x2={LOG.x - 2}
      y2={LINE_Y}
      className={styles.chromeDash}
      {...fine}
    />
    <rect
      x={LOG.x}
      y={LOG.y}
      width={LOG.w}
      height={LOG.h}
      className={styles.chromeStroke}
      {...stroke}
    />
    {CALLS.map((_, i) => (
      <rect
        key={i}
        x={slotX(i) - SLOT / 2}
        y={LINE_Y - SLOT / 2}
        width={SLOT}
        height={SLOT}
        className={styles.chromeSlot}
        {...fine}
      />
    ))}
    <rect
      x={PROVIDER.x}
      y={PROVIDER.y}
      width={PROVIDER.w}
      height={PROVIDER.h}
      className={styles.chromeFrame}
      {...fine}
    />
    <polyline
      points={`${LOG.x + LOG.w + 2},${LINE_Y} ${JUNCTION_X},${LINE_Y}`}
      className={styles.chromeDash}
      {...fine}
    />
    {[TOP, BOTTOM].map((box) => (
      <polyline
        key={box.y}
        points={`${JUNCTION_X},${LINE_Y} ${JUNCTION_X},${center(box).y} ${box.x - 4},${center(box).y}`}
        className={styles.chromeDash}
        {...fine}
      />
    ))}
    <line
      x1={SHELF.x}
      y1={TOP.y + BOX.h + 2}
      x2={SHELF.x}
      y2={BOTTOM.y - 2}
      className={styles.chromeDash}
      {...fine}
    />
    <rect
      x={SHELF.x - 6}
      y={SHELF.y - 6}
      width={12}
      height={12}
      className={styles.chromeShelf}
      {...fine}
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

const SandboxBox = ({ label }: { label: string }) => (
  <div className={styles.box}>
    <div className={styles.boxHeader}>
      <span className={styles.boxDot} />
      {label}
    </div>
  </div>
);

const ARIA_LABEL =
  "Animated diagram of Hatchet sandboxes. A task running in a worker calls create, exec, kill, create from a snapshot and connect. Each call becomes a Hatchet task that the sandbox operator forwards to a sandbox on the connected provider. Killing the sandbox stops it and stores a snapshot of its workspace. A fork then starts from that snapshot with the same files, and connect restarts the original sandbox from the same snapshot.";

export const Sandboxes = ({ style }: { style?: CSSProperties }) => (
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
        <StageLabel x={LOG.x + LOG.w / 2} y={78} muted>
          tasks
        </StageLabel>
        <StageLabel x={LOG.x + LOG.w / 2} y={136}>
          hatchet
        </StageLabel>
        <StageLabel x={PROVIDER.x + PROVIDER.w / 2} y={14}>
          provider
        </StageLabel>
        <StageLabel x={SHELF.x + 21} y={SHELF.y - 3} muted>
          snapshot
        </StageLabel>
        <Flow.Token track={TOP_TRACK}>
          <SandboxBox label="sandbox" />
        </Flow.Token>
        <Flow.Token track={BOTTOM_TRACK}>
          <SandboxBox label="fork" />
        </Flow.Token>
        {SLOT_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.slot} />
          </Flow.Token>
        ))}
        {BAR_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.bar} />
          </Flow.Token>
        ))}
        <Flow.Token track={SNAPSHOT_TRACK}>
          <div className={styles.snapshot} />
        </Flow.Token>
        {COPY_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.copy} />
          </Flow.Token>
        ))}
        {[...IN_TRACKS, ...OUT_TRACKS].map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.msg} />
          </Flow.Token>
        ))}
        <Flow.Token track={TASK_TRACK}>
          <div className={styles.taskPill}>task</div>
        </Flow.Token>
        {LABEL_TRACKS.map((track, i) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.callLabel}>{CALLS[i].label}</div>
          </Flow.Token>
        ))}
      </Flow.Stage>
    </Flow.Root>
  </div>
);
