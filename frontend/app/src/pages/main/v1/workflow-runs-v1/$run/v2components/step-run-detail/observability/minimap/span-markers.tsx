import { getSpanColor } from '../utils/span-tree-utils';
import type { SpanMarker } from './minimap-types';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { cn } from '@/lib/utils';

interface SpanMarkersProps {
  markers: SpanMarker[];
  hoveredIdx: number | null;
  onHoveredIdxChange: (idx: number | null) => void;
  onTooltipPosChange: (pos: { x: number; y: number } | null) => void;
  onSpanSelect?: (span: OtelSpanTree, ancestorSpanIds: string[]) => void;
}

export function SpanMarkers({
  markers,
  hoveredIdx,
  onHoveredIdxChange,
  onTooltipPosChange,
  onSpanSelect,
}: SpanMarkersProps) {
  return (
    <>
      {markers.map((m, i) => (
        <div
          key={i}
          className={cn(
            'absolute inset-y-[6px] flex cursor-pointer flex-col justify-center transition-[transform,opacity]',
            m.hasErrorInTree ? 'z-[3]' : 'z-[2]',
            hoveredIdx === i && 'z-[5] scale-x-150',
            !m.visible && 'opacity-[0.01]',
          )}
          style={{ left: `${m.pct * 100}%`, width: 6 }}
          onPointerDown={(e) => {
            if (onSpanSelect) {
              e.stopPropagation();
            }
          }}
          onClick={() => {
            onSpanSelect?.(m.span, m.ancestorSpanIds);
          }}
          onMouseEnter={(e) => {
            onHoveredIdxChange(i);
            onTooltipPosChange({ x: e.clientX, y: e.clientY });
          }}
          onMouseMove={(e) => {
            onTooltipPosChange({ x: e.clientX, y: e.clientY });
          }}
          onMouseLeave={() => {
            onHoveredIdxChange(null);
            onTooltipPosChange(null);
          }}
        >
          <div
            className={cn('flex-1 rounded-full', getSpanColor(m.span))}
          />
        </div>
      ))}
    </>
  );
}
