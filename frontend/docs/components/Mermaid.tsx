"use client";
import { jsx } from "react/jsx-runtime";
import { useEffect, useId, useRef, useState } from "react";

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

const BRAND_THEME = {
  primaryColor: "#0F1A3A",
  primaryTextColor: "#B8D9FF",
  primaryBorderColor: "#3392FF",
  lineColor: "#3392FF",
  secondaryColor: "#1A1035",
  secondaryTextColor: "#D585EF",
  secondaryBorderColor: "#BC46DD",
  tertiaryColor: "#0D2A15",
  tertiaryTextColor: "#86EFAC",
  tertiaryBorderColor: "#22C55E",
  background: "#02081D",
  mainBkg: "#0F1A3A",
  nodeBorder: "#3392FF",
  clusterBkg: "#0A1029",
  clusterBorder: "#1C2B4A",
  titleColor: "#B8D9FF",
  edgeLabelBackground: "#02081D",
  noteBkgColor: "#162035",
  noteTextColor: "#B8D9FF",
  noteBorderColor: "#3392FF",
  actorBorder: "#3392FF",
  actorBkg: "#0F1A3A",
  actorTextColor: "#B8D9FF",
  actorLineColor: "#3392FF",
  signalColor: "#B8D9FF",
  signalTextColor: "#B8D9FF",
  labelBoxBkgColor: "#0F1A3A",
  labelBoxBorderColor: "#3392FF",
  labelTextColor: "#B8D9FF",
  loopTextColor: "#B8D9FF",
  activationBorderColor: "#3392FF",
  activationBkgColor: "#162947",
  sequenceNumberColor: "#B8D9FF",
};

function Mermaid({ chart }: { chart: string }) {
  const id = useId();
  const [svg, setSvg] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const isVisible = useIsVisible(containerRef);

  useEffect(() => {
    if (!isVisible) return;

    const htmlElement = document.documentElement;
    const observer = new MutationObserver(renderChart);
    observer.observe(htmlElement, { attributes: true });
    renderChart();
    return () => {
      observer.disconnect();
    };

    async function renderChart() {
      const isDarkTheme =
        htmlElement.classList.contains("dark") ||
        htmlElement.attributes.getNamedItem("data-theme")?.value === "dark";

      const { default: mermaid } = await import("mermaid");
      try {
        mermaid.initialize({
          startOnLoad: false,
          securityLevel: "loose",
          fontFamily: "inherit",
          themeCSS: "margin: 1.5rem auto 0;",
          theme: isDarkTheme ? "base" : "default",
          ...(isDarkTheme ? { themeVariables: BRAND_THEME } : {}),
        });
        const { svg: rendered } = await mermaid.render(
          id.replaceAll(":", ""),
          chart.replaceAll("\\n", "\n"),
          containerRef.current ?? undefined,
        );
        setSvg(rendered);
      } catch (error) {
        console.error("Error while rendering mermaid", error);
      }
    }
  }, [chart, id, isVisible]);

  return jsx("div", {
    ref: containerRef,
    dangerouslySetInnerHTML: { __html: svg },
  });
}

export { Mermaid };
