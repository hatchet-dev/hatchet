"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";
import styles from "./flow.module.css";
import {
  FlowContextProvider,
  useFlow,
  type FlowContextValue,
  type FlowFrameCallback,
} from "./useFlow";
import {
  resolveTrack,
  sampleTrack,
  type FlowSample,
  type FlowTrack,
} from "./timeline";

// ─── Root ──────────────────────────────────────────────────────────────────

interface RootProps {
  /** Loop length in ms. The clock wraps to 0 when it reaches this. */
  duration: number;
  /**
   * Loop time (ms) rendered as the initial/static frame: server markup,
   * pre-hydration paint, and the composition shown under
   * prefers-reduced-motion. Pick a moment where the diagram tells its story.
   */
  posterTime?: number;
  /** Extra external pause source (e.g. a parent-controlled toggle). */
  paused?: boolean;
  /** Pause when the document is hidden (Page Visibility API). Default true. */
  pauseWhenHidden?: boolean;
  /** Pause when the root scrolls out of view (IntersectionObserver). Default true. */
  pauseWhenOffscreen?: boolean;
  /**
   * Presentation chrome: a progress bar + restart button row at the bottom of
   * the frame. Default true; hidden automatically under reduced motion.
   */
  controls?: boolean;
  className?: string;
  style?: CSSProperties;
  "aria-label"?: string;
  children: ReactNode;
}

const Root = ({
  duration,
  posterTime = 0,
  paused = false,
  pauseWhenHidden = true,
  pauseWhenOffscreen = true,
  controls = true,
  className,
  style,
  "aria-label": ariaLabel,
  children,
}: RootProps) => {
  const rootRef = useRef<HTMLDivElement | null>(null);
  const subscribersRef = useRef<Set<FlowFrameCallback>>(new Set());
  // The clock starts at the poster frame, so a paused Root keeps rendering
  // exactly the frame that was server-rendered rather than snapping to t=0.
  const timeRef = useRef(posterTime);

  const [isStatic, setIsStatic] = useState(false);
  const isStaticRef = useRef(false);
  const posterTimeRef = useRef(posterTime);
  posterTimeRef.current = posterTime;

  // Pause sources union — starts paused by "offscreen" until the
  // IntersectionObserver reports in.
  const pauseSourcesRef = useRef<Set<string>>(
    new Set(pauseWhenOffscreen ? ["offscreen"] : []),
  );
  const [isPaused, setIsPaused] = useState(pauseWhenOffscreen);

  const setPauseSource = useCallback((key: string, value: boolean) => {
    const set = pauseSourcesRef.current;
    const had = set.has(key);
    if (value && !had) set.add(key);
    else if (!value && had) set.delete(key);
    else return;
    setIsPaused(set.size > 0);
  }, []);

  // prefers-reduced-motion → park the clock on the poster frame.
  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    const apply = () => {
      isStaticRef.current = mq.matches;
      setIsStatic(mq.matches);
    };
    apply();
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, []);

  // Native pause primitives — single AbortController for cleanup.
  useEffect(() => {
    const el = rootRef.current;
    if (!el) return;
    const ac = new AbortController();

    if (pauseWhenHidden) {
      const onVis = () => setPauseSource("hidden", document.hidden);
      document.addEventListener("visibilitychange", onVis, {
        signal: ac.signal,
      });
      onVis();
    }

    let observer: IntersectionObserver | null = null;
    if (pauseWhenOffscreen) {
      observer = new IntersectionObserver(
        (entries) => {
          for (const entry of entries) {
            setPauseSource("offscreen", !entry.isIntersecting);
          }
        },
        { threshold: 0 },
      );
      observer.observe(el);
    }

    return () => {
      ac.abort();
      observer?.disconnect();
      setPauseSource("hidden", false);
      setPauseSource("offscreen", false);
    };
  }, [pauseWhenHidden, pauseWhenOffscreen, setPauseSource]);

  // The clock. One rAF loop broadcasts loop time to all subscribed tokens —
  // no React re-renders on the hot path. A low-frequency interval backstop
  // keeps the clock advancing in environments that starve rAF (headless
  // capture under virtual time, aggressive throttling); it shares the same
  // dt accumulation, so the two sources interleave without double-counting.
  const running = !isPaused && !paused && !isStatic;
  useEffect(() => {
    if (!running) return;
    let raf = 0;
    let last: number | null = null;
    let lastFrameAt = 0;
    const step = () => {
      const now = performance.now();
      if (last !== null) {
        // Clamp dt so a janky frame doesn't teleport tokens.
        const dt = Math.min(now - last, 100);
        timeRef.current = (timeRef.current + dt) % duration;
        for (const fn of subscribersRef.current) fn(timeRef.current);
      }
      last = now;
    };
    const onFrame = () => {
      lastFrameAt = performance.now();
      step();
      raf = requestAnimationFrame(onFrame);
    };
    raf = requestAnimationFrame(onFrame);
    const interval = window.setInterval(() => {
      // Only steps in when rAF has gone quiet.
      if (performance.now() - lastFrameAt > 40) step();
    }, 33);
    return () => {
      cancelAnimationFrame(raf);
      window.clearInterval(interval);
    };
  }, [running, duration]);

  // Static mode: broadcast the poster frame once.
  useEffect(() => {
    if (!isStatic) return;
    timeRef.current = posterTime;
    for (const fn of subscribersRef.current) fn(posterTime);
  }, [isStatic, posterTime]);

  const subscribe = useCallback((fn: FlowFrameCallback) => {
    subscribersRef.current.add(fn);
    fn(isStaticRef.current ? posterTimeRef.current : timeRef.current);
    return () => {
      subscribersRef.current.delete(fn);
    };
  }, []);

  const now = useCallback(() => timeRef.current, []);

  /**
   * Rewind the clock to 0 (posterTime under reduced motion) and broadcast
   * immediately, so the composition snaps back whether running or paused.
   */
  const resetCountRef = useRef(0);
  const resets = useCallback(() => resetCountRef.current, []);

  const reset = useCallback(() => {
    resetCountRef.current += 1;
    timeRef.current = isStaticRef.current ? posterTimeRef.current : 0;
    for (const fn of subscribersRef.current) fn(timeRef.current);
  }, []);

  // Loop-position progress for the controls row: an internal subscriber writes
  // `--flow-progress` (0..1) on the frame element — same no-render hot path as
  // tokens; the fill is a pure CSS scaleX of that property.
  useEffect(
    () =>
      subscribe((t) => {
        rootRef.current?.style.setProperty(
          "--flow-progress",
          String(t / duration),
        );
      }),
    [subscribe, duration],
  );

  const ctxValue = useMemo<FlowContextValue>(
    () => ({ duration, posterTime, isStatic, subscribe, now, reset, resets }),
    [duration, posterTime, isStatic, subscribe, now, reset, resets],
  );

  return (
    <FlowContextProvider value={ctxValue}>
      {/* Outer frame (bounding box + controls). The stage container below
          keeps role="img" and the consumer className, so interactive controls
          are NOT inside the presentational img subtree. `flow-scope` carries
          the marketing-parity design tokens (see styles/flow-tokens.css). */}
      <div
        ref={rootRef}
        className={`${styles.frame} flow-scope`}
        style={style}
        data-flow-static={isStatic || undefined}
        data-flow-paused={isPaused || paused || undefined}
      >
        <div
          role="img"
          aria-label={ariaLabel}
          className={`${styles.root} ${className ?? ""}`}
        >
          {children}
        </div>
        {controls ? (
          <div className={styles.controls}>
            <div className={styles.progressTrack} aria-hidden="true">
              <div className={styles.progressFill} />
            </div>
            <button
              type="button"
              className={styles.restart}
              onClick={reset}
              aria-label="Restart animation"
            >
              <span className={styles.restartLabel}>[&nbsp;Restart&nbsp;]</span>
            </button>
          </div>
        ) : null}
      </div>
    </FlowContextProvider>
  );
};

// ─── Stage ─────────────────────────────────────────────────────────────────

interface StageProps {
  /** Design-space width. Token x coordinates are in these units. */
  width: number;
  /** Design-space height. Sets the stage's aspect ratio with `width`. */
  height: number;
  className?: string;
  children: ReactNode;
}

/**
 * A responsive coordinate space. The stage exposes `--flow-u` (the rendered
 * size of one design unit) so tokens — and consumer CSS — can size and
 * position everything in design units. Put a `<svg viewBox="0 0 W H">` child
 * first for the static line work; it is absolutely stretched to the stage.
 */
const Stage = ({ width, height, className, children }: StageProps) => (
  <div
    className={`${styles.stage} ${className ?? ""}`}
    style={
      {
        aspectRatio: `${width} / ${height}`,
        "--flow-w": width,
      } as CSSProperties
    }
  >
    {children}
  </div>
);

// ─── Token ─────────────────────────────────────────────────────────────────

interface TokenProps {
  track: FlowTrack;
  /** Applied to the centering wrapper around `children`. */
  className?: string;
  /** The token's visual. Style it off `[data-flow-state="…"]` on the ancestor. */
  children?: ReactNode;
}

const applySample = (el: HTMLElement, s: FlowSample) => {
  el.style.visibility = s.alive ? "" : "hidden";
  if (!s.alive) return;
  el.style.setProperty("--flow-x", String(s.x));
  el.style.setProperty("--flow-y", String(s.y));
  el.style.setProperty("--flow-scale", String(s.scale));
  el.style.opacity = String(s.opacity);
  if (el.dataset.flowState !== s.state) {
    el.dataset.flowState = s.state;
  }
};

/**
 * One element driven by one track. The first render is the poster frame
 * (SSR-safe, deterministic); after mount the loop clock mutates transform /
 * opacity / `data-flow-state` imperatively — React never re-renders per frame.
 */
const Token = ({ track, className, children }: TokenProps) => {
  const { posterTime, subscribe } = useFlow();
  const ref = useRef<HTMLDivElement | null>(null);
  const resolved = useMemo(() => resolveTrack(track), [track]);

  useEffect(
    () =>
      subscribe((t) => {
        const el = ref.current;
        if (el) applySample(el, sampleTrack(resolved, t));
      }),
    [subscribe, resolved],
  );

  const initial = sampleTrack(resolved, posterTime);

  return (
    <div
      ref={ref}
      className={styles.token}
      data-flow-state={initial.state}
      style={
        {
          visibility: initial.alive ? undefined : "hidden",
          opacity: initial.opacity,
          "--flow-x": initial.x,
          "--flow-y": initial.y,
          "--flow-scale": initial.scale,
        } as CSSProperties
      }
    >
      <div className={`${styles.tokenInner} ${className ?? ""}`}>
        {children}
      </div>
    </div>
  );
};

// ─── Namespace export ──────────────────────────────────────────────────────

export const Flow = {
  Root,
  Stage,
  Token,
};
