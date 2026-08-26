/**
 * Pure timeline math for Flow.
 *
 * A track is a sorted list of keyframes on a shared loop clock. Tokens exist
 * between their first and last keyframe; position/opacity/scale interpolate
 * between keyframes, while `state` is a step function (it flips when the
 * playhead reaches the keyframe that declares it). Everything here is pure and
 * deterministic — safe to evaluate at module scope and during SSR.
 */

export type FlowEase = "linear" | "in" | "out" | "inOut" | "hold";

export interface FlowKeyframe {
  /** Time on the loop clock, in ms. Keyframes must be sorted ascending. */
  t: number;
  /** Position in stage design units (see Flow.Stage). */
  x: number;
  y: number;
  /** 0–1. Omitted → carried forward from the previous keyframe (initially 1). */
  opacity?: number;
  /** Omitted → carried forward from the previous keyframe (initially 1). */
  scale?: number;
  /**
   * Discrete state, exposed as `data-flow-state` on the token element. Applies
   * from the moment the playhead reaches this keyframe; carried forward.
   */
  state?: string;
  /**
   * Easing of the segment that *arrives at* this keyframe. Defaults to
   * "inOut". "hold" keeps the previous keyframe's values until this
   * keyframe's time (a step).
   */
  ease?: FlowEase;
}

export interface FlowTrack {
  id: string;
  keyframes: FlowKeyframe[];
}

interface ResolvedKeyframe {
  t: number;
  x: number;
  y: number;
  opacity: number;
  scale: number;
  state: string;
  ease: FlowEase;
}

export interface ResolvedFlowTrack {
  id: string;
  keyframes: ResolvedKeyframe[];
}

export interface FlowSample {
  /** False outside [first.t, last.t] — the token should not render. */
  alive: boolean;
  x: number;
  y: number;
  opacity: number;
  scale: number;
  state: string;
}

const EASE: Record<Exclude<FlowEase, "hold">, (p: number) => number> = {
  linear: (p) => p,
  in: (p) => p * p * p,
  out: (p) => 1 - Math.pow(1 - p, 3),
  inOut: (p) => (p < 0.5 ? 4 * p * p * p : 1 - Math.pow(-2 * p + 2, 3) / 2),
};

/**
 * Validates ordering (dev only) and returns the track unchanged. Purely a
 * declaration helper so timelines read as data.
 */
export const defineTrack = (id: string, keyframes: FlowKeyframe[]): FlowTrack => {
  if (process.env.NODE_ENV !== "production") {
    for (let i = 1; i < keyframes.length; i++) {
      if (keyframes[i].t < keyframes[i - 1].t) {
        throw new Error(
          `Flow track "${id}": keyframes must be sorted by t (index ${i}: ${keyframes[i].t} < ${keyframes[i - 1].t})`,
        );
      }
    }
  }
  return { id, keyframes };
};

/**
 * Carries omitted opacity/scale/state values forward so sampling only ever
 * looks at two adjacent keyframes. Do this once (memoized), not per frame.
 */
export const resolveTrack = (track: FlowTrack): ResolvedFlowTrack => {
  let opacity = 1;
  let scale = 1;
  let state = "idle";
  const keyframes = track.keyframes.map((k) => {
    opacity = k.opacity ?? opacity;
    scale = k.scale ?? scale;
    state = k.state ?? state;
    return { t: k.t, x: k.x, y: k.y, opacity, scale, state, ease: k.ease ?? "inOut" };
  });
  return { id: track.id, keyframes };
};

const DEAD: FlowSample = { alive: false, x: 0, y: 0, opacity: 0, scale: 1, state: "idle" };

/** Evaluates a resolved track at loop time `t` (ms). */
export const sampleTrack = (track: ResolvedFlowTrack, t: number): FlowSample => {
  const kfs = track.keyframes;
  if (kfs.length === 0) return DEAD;
  const first = kfs[0];
  const last = kfs[kfs.length - 1];
  if (t < first.t || t > last.t) return DEAD;

  // Linear scan — tracks are small (a dozen keyframes at most).
  let i = 0;
  while (i < kfs.length - 1 && kfs[i + 1].t <= t) i++;
  const a = kfs[i];
  if (i === kfs.length - 1) {
    return { alive: true, x: a.x, y: a.y, opacity: a.opacity, scale: a.scale, state: a.state };
  }
  const b = kfs[i + 1];
  if (b.ease === "hold") {
    return { alive: true, x: a.x, y: a.y, opacity: a.opacity, scale: a.scale, state: a.state };
  }
  const p = (t - a.t) / (b.t - a.t);
  const e = EASE[b.ease](p);
  return {
    alive: true,
    x: a.x + (b.x - a.x) * e,
    y: a.y + (b.y - a.y) * e,
    opacity: a.opacity + (b.opacity - a.opacity) * e,
    scale: a.scale + (b.scale - a.scale) * e,
    state: a.state,
  };
};

/**
 * Deterministic PRNG for timeline variation (jittered spawn times, varied
 * dwell, etc.). Never call Math.random() when building a timeline, or SSR
 * markup and loop boundaries stop being reproducible.
 */
export const createSeededRandom = (seed: string): (() => number) => {
  let s = 0;
  for (let i = 0; i < seed.length; i++) {
    s = (s << 5) - s + seed.charCodeAt(i);
    s = s & s; // 32-bit
  }
  return () => {
    s = (s * 1664525 + 1013904223) % 4294967296;
    return Math.abs(s) / 4294967296;
  };
};
