"use client";

import React from "react";
import { Tabs as BaseTabs } from "fumadocs-ui/components/ui/tabs";
import {
  TabsList,
  TabsTrigger,
  TabsContent,
} from "fumadocs-ui/components/tabs";
import { Callout } from "@/components/nextra-compat";
import { useLanguage } from "../context/LanguageContext";
import { LOGO_PATHS } from "@/lib/docs-languages";

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
function toTabLabel(name: string): React.ReactNode {
  const filename = LOGO_PATHS[name];
  if (!filename) return name;
  return (
    <span className="inline-flex items-center gap-1.5">
      <ThemedIcon src={`/${filename}`.replace(/\/+/g, "/")} />
      {name}
    </span>
  );
}

/* ── Early access ─────────────────────────────────────────── */

const EARLY_ACCESS_SDKS = ["Ruby"];

const EarlyAccessCallout: React.FC<{ language: string }> = ({ language }) => (
  <Callout type="info">
    <span className="text-sm">
      The {language} SDK is in early access, and may change. We&apos;d love your{" "}
      <a
        href="https://github.com/hatchet-dev/hatchet/issues"
        target="_blank"
        rel="noopener noreferrer"
        className="underline"
      >
        feedback
      </a>
      !
    </span>
  </Callout>
);

/* ── Component ─────────────────────────────────────────────── */

interface UniversalTabsProps {
  items: string[];
  children: React.ReactNode;
  optionKey?: string;
  variant?: "tabs" | "hidden";
}

/** Normalize item for matching (items may use "Typescript" vs "TypeScript"). */
function resolveSelectedItem(items: string[], value: string): string {
  const exact = items.find((i) => i === value);
  if (exact) return exact;
  const lower = value.toLowerCase();
  const match = items.find((i) => i.toLowerCase() === lower);
  return match ?? items[0];
}

interface Panel {
  title?: string;
  content: React.ReactNode;
}

function collectPanels(children: React.ReactNode): Panel[] {
  const panels: Panel[] = [];
  React.Children.forEach(children, (child) => {
    if (
      React.isValidElement<{ title?: string; children?: React.ReactNode }>(
        child,
      )
    ) {
      panels.push({ title: child.props.title, content: child.props.children });
    }
  });
  return panels;
}

export const UniversalTabs: React.FC<UniversalTabsProps> = ({
  items,
  children,
  optionKey = "language",
  variant = "tabs",
}) => {
  const {
    selectedLanguage,
    setSelectedLanguage,
    getSelectedOption,
    setSelectedOption,
  } = useLanguage();

  const selectedValue =
    optionKey === "language" ? selectedLanguage : getSelectedOption(optionKey);

  const resolvedValue = resolveSelectedItem(items, selectedValue);

  const handleChange = (value: string) => {
    if (optionKey === "language") {
      setSelectedLanguage(value);
    } else {
      setSelectedOption(optionKey, value);
    }
  };

  const panels = collectPanels(children);
  const contentFor = (item: string, index: number) => {
    const panel =
      panels.find(
        (p) => p.title && p.title.toLowerCase() === item.toLowerCase(),
      ) ?? panels[index];
    if (!panel) return null;
    const isEarlyAccess =
      optionKey === "language" && EARLY_ACCESS_SDKS.includes(item);
    return isEarlyAccess ? (
      <>
        <EarlyAccessCallout language={item} />
        {panel.content}
      </>
    ) : (
      panel.content
    );
  };

  if (variant === "hidden") {
    return <div>{contentFor(resolvedValue, items.indexOf(resolvedValue))}</div>;
  }

  return (
    <BaseTabs
      value={resolvedValue}
      onValueChange={handleChange}
      className="flex flex-col overflow-hidden rounded-xl border bg-fd-secondary my-4"
    >
      <TabsList>
        {items.map((item) => (
          <TabsTrigger key={item} value={item}>
            {toTabLabel(item)}
          </TabsTrigger>
        ))}
      </TabsList>
      {items.map((item, i) => (
        <TabsContent key={item} value={item}>
          {contentFor(item, i)}
        </TabsContent>
      ))}
    </BaseTabs>
  );
};

export default UniversalTabs;
