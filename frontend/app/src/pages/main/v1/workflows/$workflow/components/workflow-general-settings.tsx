import {
  EmptyState,
  FieldLabel,
  formatLimitStrategy,
} from './settings-primitives';
import { Badge } from '@/components/v1/ui/badge';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { MarkdownRenderer } from '@/components/v1/ui/markdown';
import {
  ConcurrencyScope,
  ConcurrencySetting,
  WorkflowVersion,
} from '@/lib/api';
import { formatCron } from '@/lib/cron';
import { cn } from '@/lib/utils';

export default function WorkflowGeneralSettings({
  workflow,
}: {
  workflow: WorkflowVersion;
}) {
  const hasEvents =
    !!workflow.triggers?.events && workflow.triggers.events.length > 0;
  const hasCrons =
    !!workflow.triggers?.crons && workflow.triggers.crons.length > 0;
  const hasTriggers = hasEvents || hasCrons;
  const concurrency = (workflow.v1Concurrency ?? []).filter(
    (c) => c.scope === ConcurrencyScope.WORKFLOW,
  );
  const hasIdempotency = !!workflow.idempotency;
  const hasMisc = !!workflow.sticky || workflow.defaultPriority != null;
  const hasExecution = hasIdempotency || hasMisc;

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
      {workflow.description && (
        <Section
          title="Description"
          className="text-sm leading-relaxed"
          cardClassName="md:col-span-2"
        >
          <MarkdownRenderer content={workflow.description} />
        </Section>
      )}

      {hasTriggers && (
        <Section title="Triggers">
          <TriggerSettings workflow={workflow} />
        </Section>
      )}

      {concurrency.length > 0 && (
        <Section
          title="Concurrency"
          titleRight={
            <span className="text-xs text-muted-foreground">
              {concurrency.length} limit{concurrency.length === 1 ? '' : 's'}
            </span>
          }
        >
          <ConcurrencySettings concurrency={concurrency} />
        </Section>
      )}

      {!hasExecution && (
        <Section title="Execution">
          <EmptyState message="No additional execution configuration set for this workflow." />
        </Section>
      )}

      {hasIdempotency && (
        <Section title="Idempotency">
          <IdempotencySettings workflow={workflow} />
        </Section>
      )}

      {hasMisc && (
        <Section title="Other">
          <MiscSettings workflow={workflow} />
        </Section>
      )}
    </div>
  );
}

function Section({
  title,
  titleRight,
  className,
  cardClassName,
  children,
}: {
  title: string;
  titleRight?: React.ReactNode;
  className?: string;
  cardClassName?: string;
  children: React.ReactNode;
}) {
  return (
    <Card className={cn('bg-muted/30', cardClassName)}>
      <CardHeader className="flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-base">{title}</CardTitle>
        {titleRight}
      </CardHeader>
      <CardContent className={cn('max-h-80 overflow-y-auto', className)}>
        {children}
      </CardContent>
    </Card>
  );
}

function ExpressionBlock({ code }: { code: string }) {
  return (
    <CodeHighlighter
      language="text"
      className="whitespace-pre-wrap break-words text-xs leading-relaxed"
      code={code}
      copy={false}
      maxHeight="8rem"
    />
  );
}

function TriggerSettings({ workflow }: { workflow: WorkflowVersion }) {
  return (
    <div className="flex flex-wrap gap-8">
      {workflow.triggers?.events && workflow.triggers.events.length > 0 && (
        <div>
          <FieldLabel>Events</FieldLabel>
          <div className="flex flex-wrap gap-1.5">
            {workflow.triggers.events.map((event) => (
              <Badge
                key={event.event_key}
                variant="secondary"
                className="font-mono text-xs"
              >
                {event.event_key}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {workflow.triggers?.crons && workflow.triggers.crons.length > 0 && (
        <div>
          <FieldLabel>Cron Schedules</FieldLabel>
          <div className="space-y-2">
            {workflow.triggers.crons.map((cronTrigger) => (
              <div key={cronTrigger.cron}>
                <Badge variant="secondary" className="font-mono text-xs">
                  {cronTrigger.cron}
                </Badge>
                {cronTrigger.cron && (
                  <p className="mt-1 text-xs text-muted-foreground">
                    Runs {formatCron(cronTrigger.cron)}
                  </p>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function ConcurrencySettings({
  concurrency,
}: {
  concurrency: ConcurrencySetting[];
}) {
  return (
    <div className="text-xs">
      {concurrency.map((c, i) => (
        <div
          key={i}
          className="space-y-1.5 border-border py-2.5 first:pt-0 last:pb-0 [&:not(:last-child)]:border-b"
        >
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-foreground">Max {c.maxRuns}</span>
            <span className="text-muted-foreground">·</span>
            <Badge variant="secondary" className="font-normal">
              {formatLimitStrategy(c.limitStrategy)}
            </Badge>
          </div>
          <ExpressionBlock code={c.expression} />
        </div>
      ))}
    </div>
  );
}

function IdempotencySettings({ workflow }: { workflow: WorkflowVersion }) {
  if (!workflow.idempotency) {
    return null;
  }

  return (
    <div className="flex flex-col gap-3 text-xs">
      <div>
        <FieldLabel>Idempotency Expression</FieldLabel>
        <ExpressionBlock code={workflow.idempotency.expression} />
      </div>
      <div>
        <FieldLabel>Idempotency TTL</FieldLabel>
        <div className="text-foreground">
          {workflow.idempotency.ttlMs.toLocaleString()} ms
        </div>
      </div>
    </div>
  );
}

function MiscSettings({ workflow }: { workflow: WorkflowVersion }) {
  return (
    <div className="grid grid-cols-2 gap-x-4 gap-y-3 text-xs">
      {workflow.sticky && (
        <div>
          <FieldLabel>Sticky Strategy</FieldLabel>
          <div className="font-mono text-foreground">{workflow.sticky}</div>
        </div>
      )}
      {workflow.defaultPriority != null && (
        <div>
          <FieldLabel>Default Priority</FieldLabel>
          <div className="text-foreground">{workflow.defaultPriority}</div>
        </div>
      )}
    </div>
  );
}
