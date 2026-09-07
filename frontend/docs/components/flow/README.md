# Flow

> Ported from `hatchet-marketing/src/components/Flow/` (copy-port, not shared).
> Differences from the marketing original:
>
> - `"use client"` directives on `Flow.tsx` / `useFlow.ts` (Next.js App Router).
> - `import.meta.env?.DEV` → `process.env.NODE_ENV !== "production"`.
> - The `[ Restart ]` label is an inline styled span instead of marketing's
>   `Text.Micro`.
> - `rlh` units → `calc(var(--rlh) * n)`; `--rlh` and the semantic color
>   tokens (`--fg`, `--accent-*`, …) come from `styles/flow-tokens.css`,
>   scoped under the `flow-scope` class that `Flow.Root` puts on its frame.
> - Layer order for `@layer flow-base` is declared at the top of
>   `styles/global.css` so Tailwind utilities and unlayered component CSS win.
>
> When porting a diagram from the marketing repo, run its `.module.css`
> through the same `rlh` rewrite and keep everything else identical.

A small toolkit for diagram-style animated explainers: many small tokens (squares, dots, boxes) spawning, traveling along paths, filling slots, changing state, persisting, and looping — all driven by one deterministic loop clock. Built on rAF + CSS custom properties; no per-frame React renders.

## Files

- `Flow.tsx` — compound components (`Root`, `Stage`, `Token`)
- `useFlow.ts` — clock context, `useFlow()` / `useFlowFrame()` hooks
- `timeline.ts` — pure timeline math: `defineTrack`, `resolveTrack`, `sampleTrack`, `createSeededRandom`
- `flow.module.css` — stage coordinate space + token positioning (in `@layer flow-base`, so consumer CSS wins)
- `index.ts` — barrel

## When to use Flow vs Sequence

**Sequence** is for scripted multi-*scene* demos: a handful of discrete states that crossfade on a timer (tabs, chaptered product tours). The unit of composition is a scene.

**Flow** is for continuous, timeline-driven motion of many small elements *within one composition*: tokens flowing through a queue, a scheduler fanning work out, events streaming into a log. There are no scenes — just one looping clock and per-element keyframe tracks. If you're describing your animation as "then we cut to…", use Sequence; if it's "meanwhile this square moves over here…", use Flow. They nest fine (a Flow diagram inside a Sequence scene).

## Mental model

A `Flow.Root` owns a **loop clock**: a single rAF loop that wraps at `duration` and broadcasts the current loop time to subscribers. Each `Flow.Token` is one element driven by one **track** — a sorted list of keyframes `{ t, x, y, opacity?, scale?, state?, ease? }` in the stage's design units. Position, opacity, and scale interpolate between keyframes; `state` is a step function surfaced as `data-flow-state="…"` on the token, which consumer CSS turns into color, flicker, pulse, etc.

Everything is deterministic: tracks are plain data built at module scope (use `createSeededRandom` for variation, never `Math.random()`), so server markup, the reduced-motion poster, and every loop iteration are reproducible. A loop wraps seamlessly when the composition at `t = duration` matches `t = 0` — design your tracks so it does (e.g. scroll a log up one row at the end of the loop instead of clearing it).

## Quick start

```tsx
import { Flow, defineTrack } from "~/components/Flow";

const dot = defineTrack("dot", [
  { t: 0, x: 10, y: 50, opacity: 0, state: "inflight" },
  { t: 300, x: 30, y: 50, opacity: 1, ease: "linear" },
  { t: 1500, x: 150, y: 50, ease: "out", state: "queued" },
  { t: 2500, x: 150, y: 50, ease: "hold" }, // dwell
  { t: 3000, x: 150, y: 50, opacity: 0 }, // fade out
]);

<Flow.Root duration={4000} posterTime={2000} aria-label="A token entering a queue">
  <Flow.Stage width={300} height={100}>
    <svg viewBox="0 0 300 100">{/* static line work */}</svg>
    <Flow.Token track={dot}>
      <div className={styles.square} />
    </Flow.Token>
  </Flow.Stage>
</Flow.Root>;
```

```css
.square {
  width: calc(var(--flow-u) * 9);
  height: calc(var(--flow-u) * 9);
  background: var(--fg);
}
[data-flow-state="queued"] .square {
  background: var(--accent);
}
```

## API

### `<Flow.Root>`

| Prop                 | Default | Notes                                                                                    |
| -------------------- | ------- | ---------------------------------------------------------------------------------------- |
| `duration`           | —       | Loop length in ms. The clock wraps to 0 here.                                             |
| `posterTime`         | `0`     | Loop time rendered for SSR / pre-hydration / `prefers-reduced-motion`. Pick a frame that tells the story statically. |
| `paused`             | `false` | External pause source, unioned with the built-in ones.                                    |
| `pauseWhenHidden`    | `true`  | Pause when the document is hidden (Page Visibility API).                                  |
| `pauseWhenOffscreen` | `true`  | Pause (and start paused) until the root is in view (IntersectionObserver).                |
| `controls`           | `true`  | Progress bar + restart button row at the bottom of the frame. Hidden under reduced motion. |
| `aria-label`         | —       | The stage container renders `role="img"`; describe the composition.                       |

The Root renders an outer **frame** (1px `--fg-border` bounding box, slight padding) around an inner stage container. The frame carries `data-flow-static` (reduced motion) and `data-flow-paused` (any pause source) — hooks for consumer CSS (e.g. disabling decorative `@keyframes` in static mode). The inner container carries `role="img"`, the `aria-label`, and the consumer `className` — so consumer layout CSS (grids, stacked layers) applies to the element that directly contains the stages, exactly as before the frame existed.

#### Controls

Below the stage, inside the frame, sits a thin progress bar showing loop position and a `[ RESTART ]` button. The progress fill is pure CSS: an internal subscriber writes `--flow-progress` (0–1) on the frame element each frame — no React renders — and the fill is `scaleX(var(--flow-progress))`. The restart button calls `reset()` from the Flow context: it rewinds the clock to 0 and broadcasts immediately, whether running or paused. The controls row is hidden under `prefers-reduced-motion` (the poster frame has no playhead) and can be removed entirely with `controls={false}`. Because the button is interactive, it lives outside the `role="img"` subtree.

### `<Flow.Stage width height>`

A responsive coordinate space. Sets `aspect-ratio: width / height` and exposes `--flow-u`, the rendered size of one design unit (`100cqw / width`), so consumer CSS can size things in design units: `width: calc(var(--flow-u) * 9)`. A direct `<svg viewBox="0 0 W H">` child is stretched to fill the stage — put static line work there (use `vector-effect="non-scaling-stroke"` to keep the 1.5px brand stroke weight at any size). One Root can hold several Stages sharing the same clock — that's how you keep side-by-side panels in lockstep.

### `<Flow.Token track className?>`

One element driven by one track. Children are the token's visual, centered on the track position. The first render is the poster frame (SSR-safe); after mount the clock mutates `--flow-x` / `--flow-y` / `--flow-scale` / `opacity` / `data-flow-state` imperatively.

Style states from the attribute:

```css
[data-flow-state="failed"] .square {
  background: var(--accent-1);
  animation: flicker 600ms linear both;
}
```

Keyframe animations triggered by a state flip should end at their resting value (settling tail) — same rule as Sequence's flicker recipe. Tokens don't have to move: a track whose keyframes share one position but flip `state` turns any element (a worker box, an LED) into a state machine.

### Tracks (`timeline.ts`)

| Field     | Notes                                                                                     |
| --------- | ----------------------------------------------------------------------------------------- |
| `t`       | ms on the loop clock. Keyframes must be sorted; the token only renders between the first and last `t`. |
| `x`, `y`  | Stage design units.                                                                        |
| `opacity` | Interpolated; omitted values carry forward (initially 1).                                  |
| `scale`   | Interpolated; carried forward (initially 1). Applied to the token's inner wrapper.         |
| `state`   | Step function; applies when the playhead reaches the keyframe; carried forward. Default `"idle"`. |
| `ease`    | Easing of the segment *arriving at* this keyframe: `"linear" \| "in" \| "out" \| "inOut" \| "hold"` (default `"inOut"`). `"hold"` keeps the previous keyframe's values until this time — use a `hold` keyframe to dwell somewhere. |

- `defineTrack(id, keyframes)` — declaration helper; validates ordering in dev.
- `resolveTrack(track)` / `sampleTrack(resolved, t)` — the pure evaluator, exported for tests or custom rendering via `useFlowFrame`.
- `createSeededRandom(seed)` — LCG PRNG for deterministic variation (jittered timings, varied dwell). Same family as the blog asset generator.

## Hooks

```ts
const { duration, posterTime, isStatic, subscribe, now, reset } = useFlow();

// Per-frame callback (called once immediately, then every frame while running):
useFlowFrame((t) => {
  /* custom imperative work */
});
```

## Stage aspect ratio (site convention)

Half-column marketing stages standardize on **320 design units wide with a height between 200 and 240** — an aspect ratio between 8:5 (1.6) and 4:3 (1.33). Inside the band, pick whatever serves the composition; outside it, stages read as oddly thin or push the paired text column around. Full-width stages may go wider, but the height band still applies.

## Reduced motion

Under `prefers-reduced-motion: reduce` the clock parks at `posterTime` and broadcasts it once, so the diagram renders a meaningful static composition rather than an empty or frozen-mid-tween frame. Consumer CSS should also gate decorative `@keyframes` behind `[data-flow-static]` (belt) and `@media (prefers-reduced-motion: reduce)` (suspenders — covers pre-hydration paint).

## Performance notes

- One rAF loop per Root regardless of token count; tokens update via inline style/custom-property writes, never React state. An interval backstop (33ms, only active when rAF goes quiet) keeps the clock advancing in environments that starve rAF, e.g. headless capture under `--virtual-time-budget`.
- `data-flow-state` is only written when it changes, so CSS animations keyed off it aren't restarted every frame.
- Frame delta is clamped (64ms) so a janky frame can't teleport tokens.
