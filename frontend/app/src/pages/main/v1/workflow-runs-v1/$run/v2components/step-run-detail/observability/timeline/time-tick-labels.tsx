import { formatDuration } from '../utils/format-utils';

export function TimeTickLabels({
  ticks,
  totalMs,
  offsetMs = 0,
}: {
  ticks: number[];
  totalMs: number;
  offsetMs?: number;
}) {
  return (
    <div className="relative h-5 overflow-hidden">
      {ticks.map((t) => {
        if (t >= totalMs || t / totalMs > 0.85) {
          return null;
        }
        return (
          <div
            key={t}
            className="absolute flex h-full items-center"
            style={{
              left: `${totalMs > 0 ? (t / totalMs) * 100 : 0}%`,
            }}
          >
            <span className="whitespace-nowrap font-mono text-xs uppercase tracking-wider text-muted-foreground">
              {formatDuration(t + offsetMs)}
            </span>
          </div>
        );
      })}
      <div className="absolute right-0 z-10 flex h-full items-center">
        <span className="whitespace-nowrap rounded-sm bg-background px-1 font-mono text-xs uppercase tracking-wider text-muted-foreground">
          {formatDuration(totalMs + offsetMs)}
        </span>
      </div>
    </div>
  );
}
