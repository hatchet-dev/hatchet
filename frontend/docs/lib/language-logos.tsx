import React from "react";
import { useRouter } from "next/router";

/** Maps highlight language abbreviation to logo filename in public/ */
const LOGO_BY_LANG: Record<string, string> = {
  py: "python-logo.svg",
  ts: "typescript-logo.svg",
  go: "go-logo.svg",
  rb: "ruby-logo.svg",
};

/** Renders an SVG as a CSS mask filled with currentColor (works in light + dark mode). */
function ThemedIcon({ src, size = 16 }: { src: string; size?: number }) {
  return (
    <span
      style={
        {
          display: "inline-block",
          width: size,
          height: size,
          flexShrink: 0,
          backgroundColor: "currentColor",
          WebkitMaskImage: `url(${src})`,
          WebkitMaskSize: "contain",
          WebkitMaskRepeat: "no-repeat",
          WebkitMaskPosition: "center",
          maskImage: `url(${src})`,
          maskSize: "contain",
          maskRepeat: "no-repeat",
          maskPosition: "center",
        } as React.CSSProperties
      }
    />
  );
}

/** Renders the language logo if we have one for the given language abbrev (py, ts, go, rb). */
export function LanguageLogo({
  language,
  className,
  size = 16,
}: {
  language: string;
  className?: string;
  size?: number;
}) {
  const router = useRouter();
  const basePath = router.basePath || "";
  const filename = LOGO_BY_LANG[language?.toLowerCase()];
  if (!filename) return null;
  const src = `${basePath}/${filename}`.replace(/\/+/g, "/");
  return (
    <span className={className} aria-hidden>
      <ThemedIcon src={src} size={size} />
    </span>
  );
}
