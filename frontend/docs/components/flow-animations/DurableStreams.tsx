"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  createSeededRandom,
  defineTrack,
  type FlowKeyframe,
  type FlowTrack,
} from "@/components/flow";
import styles from "./durablestreams.module.css";

/**
 * Flow animation: durable streams (docs.hatchet.run/v1/durable-streams). A
 * task publishes messages that are appended to a persisted topic log in
 * Hatchet. A live reader receives each message as it lands. Midway through
 * the run a late reader connects, replays the log from its cursor in a quick
 * burst, then catches up and tails live alongside the first reader. Each
 * reader's cursor tracks its own position on the log.
 *
 * All tracks are plain data at module scope (seeded, deterministic): server
 * markup, the reduced-motion poster (mid catch-up), and every loop iteration
 * are reproducible.
 */

const STAGE_W = 320;
const STAGE_H = 220;

const WORKER = { x: 14, y: 76, w: 76, h: 68 };
const TASK = { x: 52, y: 110 };
const STREAM_Y = 110;

const LOG = { x: 114, y: 100, w: 92, h: 20 };
const SLOTS = 8;
const SLOT = 8;
const SLOT_PITCH = 11;
const slotX = (i: number) => LOG.x + 7.5 + i * SLOT_PITCH;
const CURSOR_A_Y = 94;
const CURSOR_B_Y = 126;

const JUNCTION_X = 224;
const READER = { w: 70, h: 44 };
const READER_A = { x: 236, y: 40 };
const READER_B = { x: 236, y: 136 };
const center = (r: { x: number; y: number }) => ({
  x: r.x + READER.w / 2,
  y: r.y + READER.h / 2,
});
const barX = (i: number) => READER_A.x + 8.5 + i * 7.5;
const BAR_DY = 31;

const RUN_AT = 700;
const EMIT0 = 1100;
/** Design units per ms; constant so tail slots take visibly longer to reach. */
const SPEED = 0.13;
const DEPART_DELAY = 120;
const REPLAY_GAP = 170;

type Pt = [number, number];

const pathLength = (pts: Pt[]) =>
  pts
    .slice(1)
    .reduce(
      (sum, [x, y], i) => sum + Math.hypot(x - pts[i][0], y - pts[i][1]),
      0,
    );

/** Constant-speed keyframes along a polyline, fading in on the first leg and out on arrival. */
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
  [60, STREAM_Y],
  [72, STREAM_Y],
  [slotX(i), STREAM_Y],
];

const outPath = (reader: { x: number; y: number }): Pt[] => {
  const y = center(reader).y;
  return [
    [204, STREAM_Y],
    [212, STREAM_Y],
    [JUNCTION_X, STREAM_Y],
    [JUNCTION_X, y],
    [reader.x - 2, y],
  ];
};

const rng = createSeededRandom("durable-streams");

const EMITS: number[] = [];
{
  let t = EMIT0;
  for (let i = 0; i < SLOTS; i++) {
    EMITS.push(Math.round(t));
    t += 620 + rng() * 160;
  }
}
const STORED = EMITS.map((e, i) => arrival(e, inPath(i)));

const DEPART_A = STORED.map((s) => s + DEPART_DELAY);
const LAND_A = DEPART_A.map((d) => arrival(d, outPath(READER_A)));

/** The late reader connects with half the log already written. */
const JOIN = STORED[SLOTS / 2 - 1] + 350;

/** Replays back-to-back until it reaches the live tail, then departs with reader A. */
const DEPART_B: number[] = [];
const REPLAYED: boolean[] = [];
{
  let prev = JOIN + 200 - REPLAY_GAP;
  for (let i = 0; i < SLOTS; i++) {
    const live = STORED[i] + DEPART_DELAY;
    const next = Math.max(live, prev + REPLAY_GAP);
    DEPART_B.push(next);
    REPLAYED.push(next > live);
    prev = next;
  }
}
const LAND_B = DEPART_B.map((d) => arrival(d, outPath(READER_B)));
const LAST_REPLAYED = REPLAYED.lastIndexOf(true);
const LIVE_AT = LAND_B[LAST_REPLAYED];

const DONE_AT = EMITS[SLOTS - 1] + 300;
const FADE_AT = Math.max(LAND_A[SLOTS - 1], LAND_B[SLOTS - 1]) + 1400;
const DURATION = Math.ceil((FADE_AT + 700) / 100) * 100;
/** Static frame: late reader mid catch-up, replayed messages in flight. */
const POSTER_TIME = DEPART_B[1] + 200;

const byTime = (a: FlowKeyframe, b: FlowKeyframe) => a.t - b.t;

const fadeOut = (x: number, y: number): FlowKeyframe[] => [
  { t: FADE_AT, x, y, ease: "hold" },
  { t: FADE_AT + 450, x, y, opacity: 0, ease: "linear" },
];

const IN_TRACKS: FlowTrack[] = EMITS.map((e, i) =>
  defineTrack(`in-${i}`, alongPath(e, inPath(i), "live")),
);

const OUT_A_TRACKS: FlowTrack[] = DEPART_A.map((d, i) =>
  defineTrack(`out-a-${i}`, alongPath(d, outPath(READER_A), "live")),
);

const OUT_B_TRACKS: FlowTrack[] = DEPART_B.map((d, i) =>
  defineTrack(
    `out-b-${i}`,
    alongPath(d, outPath(READER_B), REPLAYED[i] ? "replay" : "live"),
  ),
);

/** Log slots land fresh, settle, and blip when the late reader replays them. */
const SLOT_TRACKS: FlowTrack[] = STORED.map((s, i) => {
  const x = slotX(i);
  const y = STREAM_Y;
  const kfs: FlowKeyframe[] = [
    { t: s - 40, x, y, opacity: 0, state: "fresh" },
    { t: s, x, y, opacity: 1, ease: "linear" },
    { t: s + 500, x, y, state: "stored", ease: "hold" },
  ];
  if (REPLAYED[i]) {
    kfs.push(
      { t: DEPART_B[i], x, y, state: "read", ease: "hold" },
      { t: DEPART_B[i] + 300, x, y, state: "stored", ease: "hold" },
    );
  }
  return defineTrack(`slot-${i}`, [...kfs.sort(byTime), ...fadeOut(x, y)]);
});

const barTracks = (
  id: string,
  lands: number[],
  reader: { x: number; y: number },
  replayed: boolean[],
) =>
  lands.map((land, i) => {
    const x = barX(i);
    const y = reader.y + BAR_DY;
    return defineTrack(`${id}-${i}`, [
      {
        t: land - 40,
        x,
        y,
        opacity: 0,
        state: replayed[i] ? "replay" : "fresh",
      },
      { t: land, x, y, opacity: 1, ease: "linear" },
      { t: land + 550, x, y, state: "settled", ease: "hold" },
      ...fadeOut(x, y),
    ]);
  });

const BAR_A_TRACKS = barTracks(
  "bar-a",
  LAND_A,
  READER_A,
  LAND_A.map(() => false),
);
const BAR_B_TRACKS = barTracks("bar-b", LAND_B, READER_B, REPLAYED);

/** A reader's cursor steps to each slot as that slot is delivered to it. */
const cursorTrack = (id: string, departs: number[], y: number) => {
  const kfs: FlowKeyframe[] = [
    { t: departs[0] - 200, x: slotX(0), y, opacity: 0 },
    { t: departs[0], x: slotX(0), y, opacity: 1, ease: "linear" },
  ];
  for (let i = 1; i < departs.length; i++) {
    kfs.push(
      { t: departs[i], x: slotX(i - 1), y, ease: "hold" },
      { t: departs[i] + 160, x: slotX(i), y, ease: "out" },
    );
  }
  return defineTrack(id, [...kfs, ...fadeOut(slotX(departs.length - 1), y)]);
};

const CURSOR_A_TRACK = cursorTrack("cursor-a", DEPART_A, CURSOR_A_Y);
const CURSOR_B_TRACK = cursorTrack("cursor-b", DEPART_B, CURSOR_B_Y);

const TASK_TRACK = defineTrack("task", [
  { t: 0, x: TASK.x, y: TASK.y, state: "idle" },
  { t: RUN_AT, x: TASK.x, y: TASK.y, state: "running", ease: "hold" },
  { t: DONE_AT, x: TASK.x, y: TASK.y, state: "done", ease: "hold" },
  { t: FADE_AT + 250, x: TASK.x, y: TASK.y, state: "idle", ease: "hold" },
  { t: DURATION, x: TASK.x, y: TASK.y, ease: "hold" },
]);

const READER_A_TRACK = defineTrack("reader-a", [
  { ...center(READER_A), t: 0, state: "live" },
  { ...center(READER_A), t: DURATION, ease: "hold" },
]);

const READER_B_TRACK = defineTrack("reader-b", [
  { ...center(READER_B), t: 0, state: "off" },
  { ...center(READER_B), t: JOIN, state: "catchup", ease: "hold" },
  { ...center(READER_B), t: LIVE_AT, state: "live", ease: "hold" },
  { ...center(READER_B), t: FADE_AT + 250, state: "off", ease: "hold" },
  { ...center(READER_B), t: DURATION, ease: "hold" },
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
      y={STREAM_Y - 2}
      width={4}
      height={4}
      className={styles.chromeFill}
    />
    <line
      x1={102}
      y1={STREAM_Y}
      x2={LOG.x - 2}
      y2={STREAM_Y}
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
    {Array.from({ length: SLOTS }, (_, i) => (
      <rect
        key={i}
        x={slotX(i) - SLOT / 2}
        y={STREAM_Y - SLOT / 2}
        width={SLOT}
        height={SLOT}
        className={styles.chromeSlot}
        {...fine}
      />
    ))}
    <polyline
      points={`${LOG.x + LOG.w + 2},${STREAM_Y} ${JUNCTION_X},${STREAM_Y}`}
      className={styles.chromeDash}
      {...fine}
    />
    {[READER_A, READER_B].map((reader) => (
      <polyline
        key={reader.y}
        points={`${JUNCTION_X},${STREAM_Y} ${JUNCTION_X},${center(reader).y} ${reader.x - 4},${center(reader).y}`}
        className={styles.chromeDash}
        {...fine}
      />
    ))}
  </svg>
);

const StageLabel = ({
  x,
  y,
  muted = false,
  code = false,
  children,
}: {
  x: number;
  y: number;
  muted?: boolean;
  code?: boolean;
  children: ReactNode;
}) => (
  <div
    className={`${styles.stageLabel} ${muted ? styles.labelMuted : ""} ${
      code ? styles.labelCode : ""
    }`}
    style={{
      left: `calc(var(--flow-u) * ${x})`,
      top: `calc(var(--flow-u) * ${y})`,
    }}
  >
    {children}
  </div>
);

const ReaderBox = ({ label }: { label: string }) => (
  <div className={styles.reader}>
    <div className={styles.readerHeader}>
      <span className={styles.readerDot} />
      {label}
    </div>
  </div>
);

const ARIA_LABEL =
  "Animated diagram of Hatchet durable streams: a task running in a worker publishes messages that are appended to a persisted topic log in Hatchet. A live reader receives each message as it is written. Partway through the run a late reader connects, replays the messages already in the log from its cursor, catches up, and then receives new messages live alongside the first reader.";

export const DurableStreams = ({ style }: { style?: CSSProperties }) => (
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
        <StageLabel x={TASK.x} y={126} muted code>
          publish()
        </StageLabel>
        <StageLabel x={LOG.x + LOG.w / 2} y={78} muted>
          topic
        </StageLabel>
        <StageLabel x={LOG.x + LOG.w / 2} y={136}>
          hatchet
        </StageLabel>
        <Flow.Token track={READER_A_TRACK}>
          <ReaderBox label="reader" />
        </Flow.Token>
        <Flow.Token track={READER_B_TRACK}>
          <ReaderBox label="late reader" />
        </Flow.Token>
        {SLOT_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.slot} />
          </Flow.Token>
        ))}
        {[...BAR_A_TRACKS, ...BAR_B_TRACKS].map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.bar} />
          </Flow.Token>
        ))}
        {[...IN_TRACKS, ...OUT_A_TRACKS, ...OUT_B_TRACKS].map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.msg} />
          </Flow.Token>
        ))}
        <Flow.Token track={CURSOR_A_TRACK}>
          <div className={`${styles.cursor} ${styles.cursorDown}`} />
        </Flow.Token>
        <Flow.Token track={CURSOR_B_TRACK}>
          <div className={`${styles.cursor} ${styles.cursorUp}`} />
        </Flow.Token>
        <Flow.Token track={TASK_TRACK}>
          <div className={styles.taskPill}>task</div>
        </Flow.Token>
      </Flow.Stage>
    </Flow.Root>
  </div>
);
