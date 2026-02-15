import React from "react";
import { Tabs } from "nextra/components";
import { useLanguage } from "../context/LanguageContext";

/* ── Logo map ──────────────────────────────────────────────── */

const LOGO_MAP: Record<string, string> = {
  Python: "/python-logo.svg",
  "Python-Sync": "/python-logo.svg",
  "Python-Async": "/python-logo.svg",
  Typescript: "/typescript-logo.svg",
  TypeScript: "/typescript-logo.svg",
  Go: "/go-logo.svg",
};

const tabLabelStyle: React.CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  gap: "6px",
};

/** Renders an SVG as a CSS mask filled with currentColor (works in light + dark mode). */
function ThemedIcon({ src }: { src: string }) {
  return (
    <span
      style={
        {
          display: "inline-block",
          width: 16,
          height: 16,
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

/** Returns a logo-enhanced label if a logo exists, otherwise the plain string. */
function toTabLabel(name: string): string | React.ReactElement {
  const logo = LOGO_MAP[name];
  if (!logo) return name;
  return (
    <span style={tabLabelStyle}>
      <ThemedIcon src={logo} />
      {name}
    </span>
  );
}

/* ── Component ─────────────────────────────────────────────── */

interface UniversalTabsProps {
  items: string[];
  children: React.ReactNode;
  optionKey?: string;
}

export const UniversalTabs: React.FC<UniversalTabsProps> = ({
  items,
  children,
  optionKey = "language",
}) => {
  const {
    selectedLanguage,
    setSelectedLanguage,
    getSelectedOption,
    setSelectedOption,
  } = useLanguage();

  const selectedValue =
    optionKey === "language" ? selectedLanguage : getSelectedOption(optionKey);

  const handleChange = (index: number) => {
    if (optionKey === "language") {
      setSelectedLanguage(items[index]);
    } else {
      setSelectedOption(optionKey, items[index]);
    }
  };

  const tabLabels = items.map(toTabLabel);

  return (
    <Tabs
      items={tabLabels}
      selectedIndex={
        items.includes(selectedValue) ? items.indexOf(selectedValue) : 0
      }
      onChange={handleChange}
    >
      {children}
    </Tabs>
  );
};

export default UniversalTabs;
