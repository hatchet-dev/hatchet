import { EmptyState, FieldLabel } from './settings-primitives';
import { Badge } from '@/components/v1/ui/badge';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { MarkdownRenderer } from '@/components/v1/ui/markdown';
import {
  ConcurrencyLimitStrategy,
  ConcurrencyScope,
  ConcurrencySetting,
  WorkflowVersion,
} from '@/lib/api';
import { formatCron } from '@/lib/cron';

function formatLimitStrategy(strategy: ConcurrencyLimitStrategy): string {
  switch (strategy) {
    case ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS:
      return 'Cancel In Progress';
    case ConcurrencyLimitStrategy.DROP_NEWEST:
      return 'Drop Newest';
    case ConcurrencyLimitStrategy.QUEUE_NEWEST:
      return 'Queue Newest';
    case ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN:
      return 'Group Round Robin';
    default: {
      const exhaustiveCheck: never = strategy;
      return exhaustiveCheck;
    }
  }
}

function formatScope(scope: ConcurrencyScope): string {
  switch (scope) {
    case ConcurrencyScope.WORKFLOW:
      return 'Workflow';
    case ConcurrencyScope.TASK:
      return 'Task';
    default: {
      const exhaustiveCheck: never = scope;
      return exhaustiveCheck;
    }
  }
}

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
  const concurrency = workflow.v1Concurrency ?? [];
  const hasExecution =
    !!workflow.idempotency ||
    !!workflow.sticky ||
    !!workflow.defaultPriority ||
    !!workflow.scheduleTimeout;

  return (
    <div className="flex flex-col gap-6 px-1">
      <div className="flex flex-wrap items-start gap-x-10 gap-y-8">
        {workflow.description && (
          <Section
            title="Description"
            className="min-w-[320px] flex-[2] text-sm leading-relaxed"
          >
            <MarkdownRenderer content={workflow.description} />
          </Section>
        )}

        {hasTriggers && (
          <Section title="Triggers" className="min-w-[240px] flex-1">
            <TriggerSettings workflow={workflow} />
          </Section>
        )}

        <Section
          title="Concurrency"
          className="min-w-[280px] flex-1"
          titleRight={
            concurrency.length > 0 ? (
              <span className="text-xs text-muted-foreground">
                {concurrency.length} limit{concurrency.length === 1 ? '' : 's'}
              </span>
            ) : undefined
          }
        >
          {concurrency.length > 0 ? (
            <ConcurrencySettings concurrency={concurrency} />
          ) : (
            <EmptyState message="No concurrency limits configured for this workflow." />
          )}
        </Section>

        <Section title="Execution" className="min-w-[280px] flex-1">
          {hasExecution ? (
            <ExecutionSettings workflow={workflow} />
          ) : (
            <EmptyState message="No additional execution configuration set for this workflow." />
          )}
        </Section>
      </div>
    </div>
  );
}

function Section({
  title,
  titleRight,
  className,
  children,
}: {
  title: string;
  titleRight?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={className}>
      <div className="mb-3 flex items-baseline justify-between gap-2 border-b border-border pb-2">
        <h3 className="text-[15px] font-semibold text-foreground">{title}</h3>
        {titleRight}
      </div>
      {children}
    </div>
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
          key={`${c.scope}-${c.stepReadableId ?? 'workflow'}-${i}`}
          className="space-y-1.5 border-border py-2.5 first:pt-0 last:pb-0 [&:not(:last-child)]:border-b"
        >
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono font-semibold text-foreground">
              {c.stepReadableId || formatScope(c.scope)}
            </span>
            <span className="text-muted-foreground">·</span>
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

function ExecutionSettings({ workflow }: { workflow: WorkflowVersion }) {
  return (
    <div className="grid grid-cols-2 gap-x-4 gap-y-3 text-xs">
      {workflow.idempotency && (
        <div className="col-span-2">
          <FieldLabel>Idempotency Expression</FieldLabel>
          <ExpressionBlock code={workflow.idempotency.expression} />
        </div>
      )}
      {workflow.idempotency && (
        <div>
          <FieldLabel>Idempotency TTL</FieldLabel>
          <div className="text-foreground">
            {workflow.idempotency.ttlMs.toLocaleString()} ms
          </div>
        </div>
      )}
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
      {workflow.scheduleTimeout && (
        <div>
          <FieldLabel>Schedule Timeout</FieldLabel>
          <div className="font-mono text-foreground">
            {workflow.scheduleTimeout}
          </div>
        </div>
      )}
    </div>
  );
}
