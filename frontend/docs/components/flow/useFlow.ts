"use client";

import { createContext, useContext, useEffect, useRef } from "react";

/** Called with the current loop time (ms) once per animation frame. */
export type FlowFrameCallback = (t: number) => void;

export interface FlowContextValue {
  /** Loop length in ms. */
  duration: number;
  /** Loop time rendered for SSR, reduced motion, and no-JS. */
  posterTime: number;
  /** True when prefers-reduced-motion is on — the clock is parked at posterTime. */
  isStatic: boolean;
  /**
   * Registers a per-frame callback. Invoked immediately with the current loop
   * time, then once per frame while the clock runs. Returns an unsubscribe.
   */
  subscribe: (fn: FlowFrameCallback) => () => void;
  /** Current loop time (ms) — imperative read, does not trigger renders. */
  now: () => number;
  /**
   * Rewinds the clock to 0 (posterTime under reduced motion) and broadcasts
   * immediately — works whether the clock is running or paused.
   */
  reset: () => void;
  /**
   * Number of reset() calls so far — imperative read. Lets frame subscribers
   * tell a manual rewind apart from a natural loop wrap (both jump t backward).
   */
  resets: () => number;
}

const FlowContext = createContext<FlowContextValue | null>(null);

export const FlowContextProvider = FlowContext.Provider;

export const useFlow = (): FlowContextValue => {
  const ctx = useContext(FlowContext);
  if (!ctx) {
    throw new Error("useFlow must be used within <Flow.Root>");
  }
  return ctx;
};

/**
 * Runs `fn(t)` every frame while the loop clock is running (and once
 * immediately on mount / static poster). For custom per-frame work that
 * `<Flow.Token>` doesn't cover.
 */
export const useFlowFrame = (fn: FlowFrameCallback): void => {
  const { subscribe } = useFlow();
  const fnRef = useRef(fn);
  fnRef.current = fn;
  useEffect(() => subscribe((t) => fnRef.current(t)), [subscribe]);
};
