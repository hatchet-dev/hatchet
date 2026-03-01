import React from "react";
import { useRouter } from "next/router";
import { Callout, Tabs } from "nextra/components";
import { useLanguage } from "../context/LanguageContext";

/* ── Logo map ──────────────────────────────────────────────── */

const LOGO_PATHS: Record<string, string> = {
  Python: "python-logo.svg",
  "Python-Sync": "python-logo.svg",
  "Python-Async": "python-logo.svg",
  Typescript: "typescript-logo.svg",
  TypeScript: "typescript-logo.svg",
  Go: "go-logo.svg",
  Ruby: "ruby-logo.svg",
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
function toTabLabel(name: string, basePath: string): string | React.ReactElement {
  const filename = LOGO_PATHS[name];
  if (!filename) return name;
  const src = `${basePath}/${filename}`.replace(/\/+/g, "/");
  return (
    <span style={tabLabelStyle}>
      <ThemedIcon src={src} />
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

export const UniversalTabs: React.FC<UniversalTabsProps> = ({
  items,
  children,
  optionKey = "language",
  variant = "tabs",
}) => {
  const router = useRouter();
  const basePath = router.basePath || "";

  const {
    selectedLanguage,
    setSelectedLanguage,
    getSelectedOption,
    setSelectedOption,
  } = useLanguage();

  const selectedValue =
    optionKey === "language" ? selectedLanguage : getSelectedOption(optionKey);

  const resolvedValue = resolveSelectedItem(items, selectedValue);

  const handleChange = (index: number) => {
    if (optionKey === "language") {
      setSelectedLanguage(items[index]);
    } else {
      setSelectedOption(optionKey, items[index]);
    }
  };

  const tabLabels = items.map((item) => toTabLabel(item, basePath));

  // Inject early access callout into SDK tabs that are in early access
  const processedChildren =
    optionKey === "language"
      ? React.Children.map(children, (child) => {
          if (
            React.isValidElement<{
              title?: string;
              children?: React.ReactNode;
            }>(child) &&
            child.props.title &&
            EARLY_ACCESS_SDKS.includes(child.props.title)
          ) {
            return React.cloneElement(child, {
              children: (
                <>
                  <EarlyAccessCallout language={child.props.title} />
                  {child.props.children}
                </>
              ),
            });
          }
          return child;
        })
      : children;

  if (variant === "hidden") {
    const childrenByItem = new Map<string, React.ReactNode>();
    React.Children.forEach(processedChildren, (child) => {
      if (
        React.isValidElement<{ title?: string; children?: React.ReactNode }>(
          child
        ) &&
        child.props.title
      ) {
        const key = resolveSelectedItem(items, child.props.title);
        childrenByItem.set(key, child.props.children);
      }
    });
    const selectedContent = childrenByItem.get(resolvedValue) ?? null;
    return <div>{selectedContent}</div>;
  }

  return (
    <Tabs
      items={tabLabels}
      selectedIndex={
        items.includes(resolvedValue) ? items.indexOf(resolvedValue) : 0
      }
      onChange={handleChange}
    >
      {processedChildren}
    </Tabs>
  );
};

export default UniversalTabs;
