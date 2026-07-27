import { Badge } from '@/components/v1/ui/badge';
import { useLayoutEffect, useRef, useState } from 'react';

export function TagBadge({ tag }: { tag: string }) {
  return (
    <Badge variant="secondary" className="flex-shrink-0 text-xs">
      {tag}
    </Badge>
  );
}

export function TagList({ tags }: { tags: string[] }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const measureRef = useRef<HTMLDivElement>(null);
  const [visibleCount, setVisibleCount] = useState(tags.length);

  useLayoutEffect(() => {
    const container = containerRef.current;
    const measure = measureRef.current;
    if (!container || !measure) {
      return;
    }

    const compute = () => {
      const available = container.clientWidth;
      if (available === 0) {
        return;
      }

      const items = Array.from(measure.children) as HTMLElement[];
      const GAP = 4; // gap-1 = 4px
      const ELLIPSIS_W = 20; // width reserved for "…" + gap

      let used = 0;
      let count = 0;

      for (let i = 0; i < items.length; i++) {
        const w = items[i].offsetWidth + (i > 0 ? GAP : 0);
        const isLast = i === items.length - 1;
        if (used + w + (isLast ? 0 : ELLIPSIS_W) > available) {
          break;
        }
        used += w;
        count++;
      }

      setVisibleCount(count);
    };

    compute();
    const ro = new ResizeObserver(compute);
    ro.observe(container);
    return () => ro.disconnect();
  }, [tags]);

  const clampedCount = Math.min(visibleCount, tags.length);
  const hasOverflow = clampedCount < tags.length;

  return (
    <div ref={containerRef} className="relative min-w-0 flex-1">
      {/* Invisible layer with all tags for measuring natural widths */}
      <div
        ref={measureRef}
        aria-hidden
        className="invisible pointer-events-none absolute left-0 top-0 flex flex-nowrap items-center gap-1"
      >
        {tags.map((tag, i) => (
          <TagBadge key={`m-${i}`} tag={tag} />
        ))}
      </div>
      {/* Visible layer — only tags that fit */}
      <div className="flex flex-nowrap items-center gap-1">
        {tags.slice(0, clampedCount).map((tag, i) => (
          <TagBadge key={i} tag={tag} />
        ))}
        {hasOverflow && (
          <span className="flex-shrink-0 text-xs text-muted-foreground">…</span>
        )}
      </div>
    </div>
  );
}
