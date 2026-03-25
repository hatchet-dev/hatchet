import { formatDuration } from "../utils/format-utils";

interface CursorOverlayProps {
  activePct: number;
  totalMs: number;
}

export function CursorOverlay({ activePct, totalMs }: CursorOverlayProps) {
  return (
    <>
      <div
        className="pointer-events-none absolute inset-y-0 z-[5] w-px bg-foreground/60"
        style={{ left: `${activePct * 100}%` }}
      />
      <div
        className="pointer-events-none absolute top-0.5 z-[7] whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background"
        style={{
          left: `${activePct * 100}%`,
          transform:
            activePct < 0.08
              ? "none"
              : activePct > 0.92
                ? "translateX(-100%)"
                : "translateX(-50%)",
        }}
      >
        {formatDuration(totalMs * activePct)}
      </div>
    </>
  );
}
