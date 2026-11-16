"use client";
import { useEffect, useId, useRef, useState } from "react";
import styles from "./mermaid.module.css";

function useIsVisible(ref: React.RefObject<HTMLElement | null>) {
  const [isIntersecting, setIsIntersecting] = useState(false);
  useEffect(() => {
    if (!ref.current) return;
    const observer = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) {
        observer.disconnect();
        setIsIntersecting(true);
      }
    });
    observer.observe(ref.current);
    return () => {
      observer.disconnect();
    };
  }, [ref]);
  return isIntersecting;
}

type Rgba = { r: number; g: number; b: number; a: number };

/* getComputedStyle().color returns the legacy serialization for sRGB colors:
   "rgb(r, g, b)" or "rgba(r, g, b, a)". */
function parseColor(value: string): Rgba | null {
  const match = value.match(
    /^rgba?\(\s*([\d.]+)\s*,\s*([\d.]+)\s*,\s*([\d.]+)(?:\s*,\s*([\d.]+))?\s*\)$/,
  );
  if (!match) return null;
  return {
    r: Number(match[1]),
    g: Number(match[2]),
    b: Number(match[3]),
    a: match[4] === undefined ? 1 : Number(match[4]),
  };
}

function toCss(color: Rgba): string {
  const r = Math.round(color.r);
  const g = Math.round(color.g);
  const b = Math.round(color.b);
  return color.a >= 1
    ? `rgb(${r}, ${g}, ${b})`
    : `rgba(${r}, ${g}, ${b}, ${Number(color.a.toFixed(3))})`;
}

/* Opaque blend: `top` composited over `base` at `weight` opacity (scaled by
   the top color's own alpha). Produces literal rgb() strings so mermaid's
   color math (khroma) never has to parse modern CSS color syntax. */
function tint(top: string, base: string, weight: number): string {
  const t = parseColor(top);
  const b = parseColor(base);
  if (!t || !b) return base;
  const a = weight * t.a;
  return toCss({
    r: t.r * a + b.r * (1 - a),
    g: t.g * a + b.g * (1 - a),
    b: t.b * a + b.b * (1 - a),
    a: 1,
  });
}

const COLOR_TOKENS = [
  "--fg",
  "--fg-secondary",
  "--fg-tertiary",
  "--fg-border",
  "--bg",
  "--accent",
  "--accent-1",
  "--accent-2",
  "--accent-3",
] as const;

type TokenName = (typeof COLOR_TOKENS)[number];

/* Resolve the flow-scope tokens to literal colors for the currently active
   theme. Custom properties hold hsl() expressions with nested var()s, so
   instead of parsing them we assign each one to a probe element's `color`
   and let the browser canonicalize it to rgb()/rgba(). */
function resolveTokens(scope: HTMLElement): Record<TokenName, string> {
  const probe = document.createElement("span");
  probe.style.position = "absolute";
  probe.style.visibility = "hidden";
  probe.style.pointerEvents = "none";
  scope.appendChild(probe);
  try {
    const resolved = {} as Record<TokenName, string>;
    for (const token of COLOR_TOKENS) {
      probe.style.color = `var(${token})`;
      resolved[token] = getComputedStyle(probe).color;
    }
    return resolved;
  } finally {
    probe.remove();
  }
}

function buildThemeVariables(scope: HTMLElement): Record<string, string> {
  const tokens = resolveTokens(scope);
  const fontMono = getComputedStyle(scope)
    .getPropertyValue("--font-mono")
    .trim();

  const fg = tokens["--fg"];
  const bg = tokens["--bg"];
  const accent = tokens["--accent"];
  const accent1 = tokens["--accent-1"];
  const accent2 = tokens["--accent-2"];
  const accent3 = tokens["--accent-3"];
  const line = tokens["--fg-secondary"];
  const border = tokens["--fg-border"];

  const nodeFill = tint(fg, bg, 0.05);
  const clusterFill = tint(fg, bg, 0.025);
  const noteFill = tint(accent2, bg, 0.16);
  const activationFill = tint(accent, bg, 0.16);
  const secondaryFill = tint(accent1, bg, 0.12);
  const tertiaryFill = tint(accent3, bg, 0.12);

  return {
    ...(fontMono ? { fontFamily: fontMono } : {}),
    fontSize: "14px",
    primaryColor: nodeFill,
    primaryTextColor: fg,
    primaryBorderColor: accent,
    lineColor: line,
    secondaryColor: secondaryFill,
    secondaryTextColor: fg,
    secondaryBorderColor: accent1,
    tertiaryColor: tertiaryFill,
    tertiaryTextColor: fg,
    tertiaryBorderColor: accent3,
    background: bg,
    mainBkg: nodeFill,
    nodeBorder: accent,
    clusterBkg: clusterFill,
    clusterBorder: border,
    titleColor: fg,
    edgeLabelBackground: bg,
    noteBkgColor: noteFill,
    noteTextColor: fg,
    noteBorderColor: line,
    actorBorder: accent,
    actorBkg: nodeFill,
    actorTextColor: fg,
    actorLineColor: line,
    signalColor: line,
    signalTextColor: fg,
    labelBoxBkgColor: nodeFill,
    labelBoxBorderColor: accent,
    labelTextColor: fg,
    loopTextColor: fg,
    activationBorderColor: accent,
    activationBkgColor: activationFill,
    sequenceNumberColor: bg,
  };
}

function Mermaid({ chart }: { chart: string }) {
  const id = useId();
  const [svg, setSvg] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const isVisible = useIsVisible(containerRef);

  useEffect(() => {
    if (!isVisible) return;

    let disposed = false;
    const observer = new MutationObserver(() => {
      void renderChart();
    });
    observer.observe(document.documentElement, { attributes: true });
    void renderChart();
    return () => {
      disposed = true;
      observer.disconnect();
    };

    async function renderChart() {
      const wrapper = containerRef.current;
      if (!wrapper) return;

      /* Resolve tokens synchronously before any await, so the values match
         the theme state that triggered this render. Light vs dark falls out
         automatically: `.dark .flow-scope` overrides the tokens, and the
         MutationObserver re-runs this whenever the html element's class or
         data-theme attribute flips. */
      const themeVariables = buildThemeVariables(wrapper);

      const { default: mermaid } = await import("mermaid");
      try {
        mermaid.initialize({
          startOnLoad: false,
          securityLevel: "loose",
          fontFamily: "inherit",
          theme: "base",
          themeVariables,
        });
        const { svg: rendered } = await mermaid.render(
          id.replaceAll(":", ""),
          chart.replaceAll("\\n", "\n"),
          containerRef.current ?? undefined,
        );
        if (!disposed) setSvg(rendered);
      } catch (error) {
        console.error("Error while rendering mermaid", error);
      }
    }
  }, [chart, id, isVisible]);

  return (
    <div
      ref={containerRef}
      className={`flow-scope ${styles.frame}`}
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  );
}

export { Mermaid };
