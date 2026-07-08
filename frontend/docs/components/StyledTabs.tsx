import React, { useId, useRef, useState } from "react";

interface TabProps {
  title?: string;
  children?: React.ReactNode;
}

function Tab({ children }: TabProps) {
  return <>{children}</>;
}

interface TabsProps {
  items: React.ReactNode[];
  selectedIndex?: number;
  defaultIndex?: number;
  onChange?: (index: number) => void;
  children: React.ReactNode;
}

/**
 * Drop-in replacement for nextra's <Tabs> with the segmented pill styling
 * shared with LanguageSwitcher, so tab switchers look the same on every page.
 */
export function Tabs({
  items,
  selectedIndex,
  defaultIndex = 0,
  onChange,
  children,
}: TabsProps) {
  const [internalIndex, setInternalIndex] = useState(defaultIndex);
  const index = selectedIndex ?? internalIndex;
  // MDX can emit whitespace-only text nodes between <Tabs.Tab> elements, which
  // would shift the index-based pairing of panels with `items`.
  const panels = React.Children.toArray(children).filter(
    (child) => !(typeof child === "string" && child.trim() === ""),
  );
  const baseId = useId();
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);

  const handleSelect = (i: number) => {
    setInternalIndex(i);
    onChange?.(i);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    let next: number | null = null;
    if (e.key === "ArrowRight") next = (index + 1) % items.length;
    else if (e.key === "ArrowLeft")
      next = (index - 1 + items.length) % items.length;
    else if (e.key === "Home") next = 0;
    else if (e.key === "End") next = items.length - 1;
    if (next !== null) {
      e.preventDefault();
      handleSelect(next);
      tabRefs.current[next]?.focus();
    }
  };

  return (
    <div className="mt-4">
      <div
        className="flex flex-wrap gap-1 rounded-lg border border-[hsl(var(--border))] bg-[hsl(var(--muted)/0.5)] p-1"
        role="tablist"
        onKeyDown={handleKeyDown}
      >
        {items.map((item, i) => {
          const isSelected = i === index;
          return (
            <button
              key={i}
              ref={(el) => {
                tabRefs.current[i] = el;
              }}
              type="button"
              role="tab"
              id={`${baseId}-tab-${i}`}
              aria-controls={`${baseId}-panel-${i}`}
              aria-selected={isSelected}
              tabIndex={isSelected ? 0 : -1}
              onClick={() => handleSelect(i)}
              className={`
              inline-flex items-center gap-1.5 rounded-md px-3 py-2 text-sm font-medium
              transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]
              ${
                isSelected
                  ? "bg-[hsl(var(--background))] dark:bg-[hsl(var(--muted-foreground)/0.25)] text-[hsl(var(--foreground))] shadow-sm"
                  : "text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]"
              }
            `}
            >
              {item}
            </button>
          );
        })}
      </div>
      {panels.map((panel, i) => (
        <div
          key={i}
          role="tabpanel"
          id={`${baseId}-panel-${i}`}
          aria-labelledby={`${baseId}-tab-${i}`}
          hidden={i !== index}
          className={`mt-6 rounded ${i === index ? "" : "hidden"}`}
        >
          {React.isValidElement<TabProps>(panel) ? panel.props.children : panel}
        </div>
      ))}
    </div>
  );
}

Tabs.Tab = Tab;

export default Tabs;
