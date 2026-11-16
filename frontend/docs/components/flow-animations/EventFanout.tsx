"use client";

import type { CSSProperties, ReactNode } from "react";
import {
  Flow,
  createSeededRandom,
  defineTrack,
  type FlowKeyframe,
} from "@/components/flow";
import styles from "./eventfanout.module.css";

/**
 * Flow animation: one event, many tasks (docs.hatchet.run/v1/events).
 *
 * A `user:signup` event is pushed once from the left, absorbed by the central
 * event node, and fanned out — one copy per task that declared
 * `on_events: ["user:signup"]` — along dashed connectors. Each copy becomes an
 * independent run in its task lane: it processes (accent pulse), completes
 * (green), and settles into that lane's run tally. Every push triggers all
 * three tasks, because each one subscribed to the same event key.
 *
 * Everything is seeded/deterministic and built at module scope; the loop ends
 * with the tallies dimming out while nothing is in flight, so t = DURATION
 * wraps cleanly to the empty t = 0 composition.
 */

// ─── Geometry (stage design units, 320 × 150 — flat fan-out composition) ───

const STAGE_W = 320;
const STAGE_H = 150;

const PUSH_Y = 75; // the push line and bus centerline
const SPAWN_X = 10;
const BUS_CX = 114;
const BUS_W = 28;
const BUS_H = 40;
const FAN_X = 146; // junction node where copies split toward lanes

const LANE_X0 = 194;
const LANE_W = 120;
const LANE_H = 26;
const ENTRY_X = 192; // where the diagonal meets the lane row
const PROC_X = 206; // processing socket inside the lane

const TALLY_X0 = 236;
const TALLY_PITCH = 18;
const TALLY_SLOTS = 4;
const tallySlotX = (slot: number) => TALLY_X0 + slot * TALLY_PITCH;

// ─── The event and the tasks subscribed to it ──────────────────────────────

const EVENT_KEY = "user:signup";

interface Lane {
  name: string;
  y: number;
}

/** Each task declared `on_events: ["user:signup"]`, so every push fans out
 * to all three lanes. */
const LANES: Lane[] = [
  { name: "send_welcome_email", y: 30 },
  { name: "grant_new_user_credits", y: 75 },
  { name: "reward_referral", y: 120 },
];

// ─── Timing constants (ms) ─────────────────────────────────────────────────

const TRAVEL_IN = 1250; // spawn → bus center
const ABSORB_MS = 300; // event token shrinks into the bus
const COPY_DELAY = 200; // copies depart just after absorption
const STAGGER = 140; // cascade between sibling copies of one event
const EXIT_MS = 220; // bus center → fan junction
const DIAG_MS = 620; // fan junction → lane entry
const ENTER_MS = 300; // lane entry → processing socket
const DONE_MS = 340; // green completion beat at the socket
const FLY_MS = 440; // socket → tally slot

// ─── Seeded schedule: four pushes of the same event ────────────────────────

const rng = createSeededRandom("event-fanout");

const PUSH_BASE = [500, 3700, 6900, 10100];

const EVENTS = PUSH_BASE.map((base) => ({
  push: base + Math.round(rng() * 300 - 150),
}));

interface Run {
  eventIndex: number;
  laneIndex: number;
  slot: number;
  dep: number;
  procStart: number;
  procEnd: number;
  tallyAt: number;
}

/** One run per (event, subscribed task) — the fan-out itself. */
const RUNS: Run[] = EVENTS.flatMap((ev, eventIndex) =>
  LANES.map((_, laneIndex): Run => {
    const dep = ev.push + TRAVEL_IN + COPY_DELAY + laneIndex * STAGGER;
    const procStart = dep + EXIT_MS + DIAG_MS + ENTER_MS;
    const procEnd = procStart + Math.round(1000 + rng() * 600);
    return {
      eventIndex,
      laneIndex,
      slot: eventIndex,
      dep,
      procStart,
      procEnd,
      tallyAt: procEnd + DONE_MS + FLY_MS,
    };
  }),
);

// ─── Loop bookkeeping ──────────────────────────────────────────────────────

const lastTally = Math.max(...RUNS.map((r) => r.tallyAt));
/** Wrap beat: tallies dim out at the end while nothing is in flight. */
const DURATION = Math.ceil((lastTally + 700) / 100) * 100;
/** Static frame: the 3rd push mid-fanout — three copies on the connectors,
 * earlier runs already settled into their tallies. */
const POSTER_TIME = EVENTS[2].push + 1900;

// ─── Tracks ────────────────────────────────────────────────────────────────

/** Pushed event: spawn left, ride the dashed line (key label above), fade the
 * label on approach, get absorbed into the bus. */
const EVENT_TRACKS = EVENTS.map((ev, i) => {
  const t0 = ev.push;
  const arrive = t0 + TRAVEL_IN;
  const speed = (BUS_CX - 26) / (TRAVEL_IN - 140);
  return defineTrack(`event-${i}`, [
    { t: t0, x: SPAWN_X, y: PUSH_Y, opacity: 0, state: "push" },
    { t: t0 + 140, x: 26, y: PUSH_Y, opacity: 1, ease: "linear" },
    {
      t: t0 + 560,
      x: 26 + 420 * speed,
      y: PUSH_Y,
      ease: "linear",
      state: "arriving",
    },
    { t: arrive, x: BUS_CX, y: PUSH_Y, ease: "linear", state: "absorbed" },
    {
      t: arrive + ABSORB_MS,
      x: BUS_CX,
      y: PUSH_Y,
      opacity: 0,
      scale: 0.4,
      ease: "in",
    },
  ]);
});

/** Fanned-out copy → run → completed tally cell. */
const RUN_TRACKS = RUNS.map((r) => {
  const y = LANES[r.laneIndex].y;
  const slotX = tallySlotX(r.slot);
  const settleAt = Math.min(r.tallyAt + 900, DURATION - 620);
  return defineTrack(`run-${r.eventIndex}-${r.laneIndex}`, [
    { t: r.dep, x: BUS_CX, y: PUSH_Y, opacity: 0, state: "copy" },
    { t: r.dep + EXIT_MS, x: FAN_X, y: PUSH_Y, opacity: 1, ease: "linear" },
    { t: r.dep + EXIT_MS + DIAG_MS, x: ENTRY_X, y, ease: "inOut" },
    { t: r.procStart, x: PROC_X, y, ease: "out", state: "run" },
    { t: r.procEnd, x: PROC_X, y, ease: "hold", state: "done" },
    { t: r.procEnd + DONE_MS, x: PROC_X, y, ease: "hold" },
    { t: r.tallyAt, x: slotX, y, ease: "inOut", state: "tallied" },
    { t: settleAt, x: slotX, y, ease: "hold", state: "settled" },
    { t: DURATION - 560, x: slotX, y, ease: "hold" },
    { t: DURATION - 120, x: slotX, y, opacity: 0, ease: "linear" },
  ]);
});

/** The bus border flashes accent as each event is absorbed. */
const BUS_TRACK = defineTrack("bus", [
  { t: 0, x: BUS_CX, y: PUSH_Y, state: "idle" },
  ...EVENTS.flatMap((ev): FlowKeyframe[] => [
    {
      t: ev.push + TRAVEL_IN - 40,
      x: BUS_CX,
      y: PUSH_Y,
      state: "busy",
      ease: "hold",
    },
    {
      t: ev.push + TRAVEL_IN + 340,
      x: BUS_CX,
      y: PUSH_Y,
      state: "idle",
      ease: "hold",
    },
  ]),
  { t: DURATION, x: BUS_CX, y: PUSH_Y, ease: "hold" },
]);

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
  <svg viewBox={`0 0 ${STAGE_W} ${STAGE_H}`} aria-hidden="true">
    {/* Push origin + dashed push line into the bus */}
    <rect
      x={SPAWN_X - 2}
      y={PUSH_Y - 2}
      width={4}
      height={4}
      className={styles.chromeFill}
    />
    <line
      x1={16}
      y1={PUSH_Y}
      x2={BUS_CX - BUS_W / 2 - 4}
      y2={PUSH_Y}
      className={styles.chromeDash}
      {...fine}
    />
    {/* Fan junction: stub out of the bus, then one dashed connector per
        subscribed task lane */}
    <line
      x1={BUS_CX + BUS_W / 2}
      y1={PUSH_Y}
      x2={FAN_X - 4}
      y2={PUSH_Y}
      className={styles.chromeDash}
      {...fine}
    />
    <rect
      x={FAN_X - 2}
      y={PUSH_Y - 2}
      width={4}
      height={4}
      className={styles.chromeFill}
    />
    {LANES.map((lane) => (
      <line
        key={lane.name}
        x1={FAN_X}
        y1={PUSH_Y}
        x2={ENTRY_X - 2}
        y2={lane.y}
        className={styles.chromeDash}
        {...fine}
      />
    ))}
    {/* Task lanes: row box, processing socket, tally slot ticks */}
    {LANES.map((lane) => (
      <g key={lane.name}>
        <rect
          x={LANE_X0}
          y={lane.y - LANE_H / 2}
          width={LANE_W}
          height={LANE_H}
          className={styles.laneBox}
          {...stroke}
        />
        <rect
          x={PROC_X - 4.5}
          y={lane.y - 4.5}
          width={9}
          height={9}
          className={styles.chromeSocket}
          {...fine}
        />
        {Array.from({ length: TALLY_SLOTS }, (_, s) => (
          <line
            key={s}
            x1={tallySlotX(s) - 2.5}
            y1={lane.y}
            x2={tallySlotX(s) + 2.5}
            y2={lane.y}
            className={styles.chromeTick}
            {...fine}
          />
        ))}
      </g>
    ))}
  </svg>
);

const StageLabel = ({
  x,
  y,
  anchor = "center",
  caps = false,
  muted = false,
  small = false,
  children,
}: {
  x: number;
  y: number;
  anchor?: "center" | "left" | "right";
  caps?: boolean;
  muted?: boolean;
  small?: boolean;
  children: ReactNode;
}) => (
  <div
    className={[
      styles.stageLabel,
      anchor === "left" ? styles.anchorLeft : "",
      anchor === "right" ? styles.anchorRight : "",
      caps ? styles.labelCaps : "",
      muted ? styles.labelMuted : "",
      small ? styles.labelSmall : "",
    ].join(" ")}
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
  "Animated diagram of Hatchet event fan-out: each push of the user:signup event lands in a central event node and fans out to every task that declared on_events for that key — send_welcome_email, grant_new_user_credits, and reward_referral — spawning an independent run in each that processes, completes, and lands in that task's run tally.";

export const EventFanout = ({
  className,
  style,
}: {
  className?: string;
  style?: CSSProperties;
}) => (
  <div
    className={[styles.fanout, className ?? ""].join(" ").trim()}
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
        <StageLabel x={22} y={PUSH_Y + 10} caps muted>
          push
        </StageLabel>
        <StageLabel x={BUS_CX} y={PUSH_Y - BUS_H / 2 - 12} caps>
          events
        </StageLabel>
        {LANES.map((lane) => (
          <StageLabel
            key={lane.name}
            x={LANE_X0 + 2}
            y={lane.y - 23}
            anchor="left"
          >
            {lane.name}
          </StageLabel>
        ))}
        {LANES.map((lane) => (
          <StageLabel
            key={lane.name}
            x={LANE_X0 + LANE_W - 2}
            y={lane.y - 22}
            anchor="right"
            muted
            small
          >
            {EVENT_KEY}
          </StageLabel>
        ))}
        <Flow.Token track={BUS_TRACK}>
          <div
            className={styles.busBox}
            style={{
              width: `calc(var(--flow-u) * ${BUS_W})`,
              height: `calc(var(--flow-u) * ${BUS_H})`,
            }}
          />
        </Flow.Token>
        {EVENTS.map((_, i) => (
          <Flow.Token key={EVENT_TRACKS[i].id} track={EVENT_TRACKS[i]}>
            <div className={styles.square}>
              <div className={styles.keyLabel}>{EVENT_KEY}</div>
            </div>
          </Flow.Token>
        ))}
        {RUN_TRACKS.map((track) => (
          <Flow.Token key={track.id} track={track}>
            <div className={styles.square} />
          </Flow.Token>
        ))}
      </Flow.Stage>
    </Flow.Root>
  </div>
);
