import { type UseCaseChoice } from './onboarding-steps-types';

// Small, consistent line-art graphics for the onboarding use-case cards, in
// the marketing site's visual language (line diagrams with a single purple
// accent node). One per use case. The linework uses currentColor so the card
// can dim/brighten it with selection; the accent stays purple.
//
// Purple = the marketing --accent-1 / dashboard --danger (#BC46DD, hsl 287 69%
// 57%), used here deliberately as the highlight color.

const PURPLE = 'hsl(287, 69%, 57%)';

type GraphicProps = { className?: string };

const svgBase = {
  viewBox: '0 0 64 48',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 1.5,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
};

// Simple task: input -> [node] -> output.
function SimpleGraphic({ className }: GraphicProps) {
  return (
    <svg {...svgBase} className={className} aria-hidden>
      <path d="M6 24h12" />
      <path d="M15 21l3 3-3 3" />
      <rect x="24" y="16" width="16" height="16" rx="3" />
      <circle cx="32" cy="24" r="2.5" fill={PURPLE} stroke="none" />
      <path d="M46 24h12" />
      <path d="M55 21l3 3-3 3" />
    </svg>
  );
}

// Scheduled / cron: a timeline with tick marks and a purple "next run" marker
// with a caret above it.
function ScheduledGraphic({ className }: GraphicProps) {
  return (
    <svg {...svgBase} className={className} aria-hidden>
      <path d="M6 32h52" />
      <path d="M16 28v8M28 28v8M40 28v8M52 28v8" opacity="0.5" />
      <path d="M40 14l-3 4h6z" fill={PURPLE} stroke="none" />
      <circle cx="40" cy="32" r="3.5" fill={PURPLE} stroke="none" />
    </svg>
  );
}

// Fan-out / parallel: one source node fanning into three parallel lanes.
function FanoutGraphic({ className }: GraphicProps) {
  return (
    <svg {...svgBase} className={className} aria-hidden>
      <circle cx="14" cy="24" r="4" fill={PURPLE} stroke="none" />
      <path d="M18 24h10l6-11h8M18 24h16M18 24h10l6 11h8" opacity="0.9" />
      <rect x="44" y="9" width="14" height="8" rx="2" />
      <rect x="44" y="20" width="14" height="8" rx="2" />
      <rect x="44" y="31" width="14" height="8" rx="2" />
    </svg>
  );
}

// Event-driven: an event pulse arriving from the left, triggering a run.
function EventGraphic({ className }: GraphicProps) {
  return (
    <svg {...svgBase} className={className} aria-hidden>
      <circle cx="12" cy="24" r="3" fill={PURPLE} stroke="none" />
      <path d="M18 18a9 9 0 010 12" opacity="0.6" />
      <path d="M22 14a15 15 0 010 20" opacity="0.35" />
      <path d="M32 24h8" />
      <path d="M37 21l3 3-3 3" />
      <rect x="42" y="16" width="16" height="16" rx="3" />
    </svg>
  );
}

// Durable / long-running: a node wrapped in a replay loop.
function DurableGraphic({ className }: GraphicProps) {
  return (
    <svg {...svgBase} className={className} aria-hidden>
      <rect x="24" y="16" width="16" height="16" rx="3" />
      <path d="M44 18a16 12 0 11-4-4" />
      <path d="M40 8l1 6-6 0" fill={PURPLE} stroke={PURPLE} />
    </svg>
  );
}

// Custom / describe your own: a dashed placeholder with a purple spark.
function CustomGraphic({ className }: GraphicProps) {
  return (
    <svg {...svgBase} className={className} aria-hidden>
      <rect x="16" y="8" width="32" height="32" rx="4" strokeDasharray="4 4" />
      <path
        d="M32 17l2.2 4.8L39 24l-4.8 2.2L32 31l-2.2-4.8L25 24l4.8-2.2z"
        fill={PURPLE}
        stroke="none"
      />
    </svg>
  );
}

export const useCaseGraphics: Record<
  UseCaseChoice,
  (props: GraphicProps) => JSX.Element
> = {
  simple: SimpleGraphic,
  scheduled: ScheduledGraphic,
  fanout: FanoutGraphic,
  event: EventGraphic,
  durable: DurableGraphic,
  custom: CustomGraphic,
};
