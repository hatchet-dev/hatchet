import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { CopyWorkflowConfigButton } from '@/components/v1/shared/copy-workflow-config';
import { Badge } from '@/components/v1/ui/badge';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Label } from '@/components/v1/ui/label';
import { MarkdownRenderer } from '@/components/v1/ui/markdown';
import {
  ConcurrencyLimitStrategy,
  ConcurrencyScope,
  WorkflowVersion,
  WorkflowVersionTask,
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
  const hasTriggers =
    (workflow.triggers?.events && workflow.triggers.events.length > 0) ||
    (workflow.triggers?.crons && workflow.triggers.crons.length > 0);

  const hasTasks = !!workflow.tasks && workflow.tasks.length > 0;
  const hasConcurrency =
    !!workflow.v1Concurrency && workflow.v1Concurrency.length > 0;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        {workflow.description ? (
          <MarkdownRenderer
            content={workflow.description}
            className="min-w-0 flex-1"
          />
        ) : (
          <div />
        )}
        <div className="flex-shrink-0">
          <CopyWorkflowConfigButton workflowConfig={workflow.workflowConfig} />
        </div>
      </div>

      {hasTriggers && (
        <SettingsSection title="Triggers">
          <TriggerSettings workflow={workflow} />
        </SettingsSection>
      )}

      {hasTasks && (
        <SettingsSection title="Tasks">
          <TaskSettings tasks={workflow.tasks ?? []} />
        </SettingsSection>
      )}

      {hasConcurrency && (
        <SettingsSection title="Concurrency">
          <ConcurrencySettings workflow={workflow} />
        </SettingsSection>
      )}

      {workflow.idempotency && (
        <SettingsSection title="Idempotency">
          <IdempotencySettings idempotency={workflow.idempotency} />
        </SettingsSection>
      )}

      <SettingsSection title="Configuration">
        <ConfigurationSettings workflow={workflow} />
      </SettingsSection>
    </div>
  );
}

function SettingsSection({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-3">
      <h3 className="border-b border-gray-200 pb-2 text-base font-semibold text-gray-900 dark:border-gray-700 dark:text-gray-100">
        {title}
      </h3>
      <div className="pl-1">{children}</div>
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <p className="text-sm italic text-gray-500 dark:text-gray-400">{message}</p>
  );
}

function FieldGroup({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-1">
      <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </Label>
      {children}
    </div>
  );
}

function TriggerSettings({ workflow }: { workflow: WorkflowVersion }) {
  if (!workflow.triggers) {
    return (
      <EmptyState message="There are no trigger settings for this workflow." />
    );
  }

  return (
    <div className="space-y-4">
      {workflow.triggers.events && workflow.triggers.events.length > 0 && (
        <FieldGroup label="Events">
          <div className="flex flex-wrap gap-1">
            {workflow.triggers.events.map((event) => (
              <Badge
                key={event.event_key}
                variant="secondary"
                className="font-mono text-sm"
              >
                {event.event_key}
              </Badge>
            ))}
          </div>
        </FieldGroup>
      )}

      {workflow.triggers.crons && workflow.triggers.crons.length > 0 && (
        <FieldGroup label="Cron Schedules">
          <div className="space-y-2">
            {workflow.triggers.crons.map((cronTrigger) => (
              <div key={cronTrigger.cron}>
                <Badge variant="secondary" className="font-mono text-sm">
                  {cronTrigger.cron}
                </Badge>
                {cronTrigger.cron && (
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    Runs {formatCron(cronTrigger.cron)}
                  </p>
                )}
              </div>
            ))}
          </div>
        </FieldGroup>
      )}
    </div>
  );
}

function TaskSettings({ tasks }: { tasks: WorkflowVersionTask[] }) {
  return (
    <SimpleTable
      data={tasks}
      rowKey={(row) => row.readableId}
      columns={[
        {
          columnLabel: 'Task',
          cellRenderer: (row) => (
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm font-medium">
                {row.readableId}
              </span>
              {row.isDurable && (
                <Badge variant="outline" className="text-xs">
                  Durable
                </Badge>
              )}
            </div>
          ),
        },
        {
          columnLabel: 'Action',
          cellRenderer: (row) => (
            <span className="whitespace-nowrap font-mono text-sm text-gray-600 dark:text-gray-400">
              {row.action}
            </span>
          ),
        },
        {
          columnLabel: 'Depends On',
          cellRenderer: (row) =>
            row.parents.length > 0 ? (
              <div className="flex flex-wrap gap-1">
                {row.parents.map((parent) => (
                  <Badge
                    key={parent}
                    variant="secondary"
                    className="font-mono text-xs"
                  >
                    {parent}
                  </Badge>
                ))}
              </div>
            ) : (
              <span className="text-sm text-gray-400">—</span>
            ),
        },
        {
          columnLabel: 'Retries',
          cellRenderer: (row) => <span className="text-sm">{row.retries}</span>,
        },
        {
          columnLabel: 'Timeout',
          cellRenderer: (row) => (
            <span className="whitespace-nowrap font-mono text-sm">
              {row.timeout || '—'}
            </span>
          ),
        },
        {
          columnLabel: 'Schedule Timeout',
          cellRenderer: (row) => (
            <span className="whitespace-nowrap font-mono text-sm">
              {row.scheduleTimeout || '—'}
            </span>
          ),
        },
        {
          columnLabel: 'Backoff',
          cellRenderer: (row) =>
            row.retryBackoffFactor ? (
              <span className="whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                x{row.retryBackoffFactor}
                {row.retryBackoffMaxSeconds
                  ? ` up to ${row.retryBackoffMaxSeconds}s`
                  : ''}
              </span>
            ) : (
              <span className="text-sm text-gray-400">—</span>
            ),
        },
        {
          columnLabel: 'Rate Limits',
          cellRenderer: (row) =>
            row.rateLimits && row.rateLimits.length > 0 ? (
              <div className="flex flex-wrap gap-1">
                {row.rateLimits.map((rl, i) => (
                  <Badge
                    key={`${rl.key ?? rl.keyExpression ?? i}`}
                    variant="secondary"
                    className="font-mono text-xs"
                  >
                    {rl.key ?? rl.keyExpression}
                  </Badge>
                ))}
              </div>
            ) : (
              <span className="text-sm text-gray-400">—</span>
            ),
        },
      ]}
    />
  );
}

function ConcurrencySettings({ workflow }: { workflow: WorkflowVersion }) {
  if (!workflow.v1Concurrency || workflow.v1Concurrency.length === 0) {
    return (
      <EmptyState message="There are no concurrency settings for this workflow." />
    );
  }

  return (
    <SimpleTable
      data={workflow.v1Concurrency
        .map((c) => ({
          stepReadableId: c.stepReadableId || 'N/A',
          ...c,
        }))
        .sort(
          (a, b) =>
            b.scope.localeCompare(a.scope) ||
            a.stepReadableId.localeCompare(b.stepReadableId),
        )}
      rowKey={(row, i) => `${row.scope}-${row.stepReadableId}-${i}`}
      columns={[
        { columnLabel: 'Scope', cellRenderer: (row) => formatScope(row.scope) },
        { columnLabel: 'Task', cellRenderer: (row) => row.stepReadableId },
        { columnLabel: 'Max', cellRenderer: (row) => row.maxRuns },
        {
          columnLabel: 'Strategy',
          cellRenderer: (row) => (
            <Badge variant="secondary">
              {formatLimitStrategy(row.limitStrategy)}
            </Badge>
          ),
        },
        {
          columnLabel: 'Expression',
          cellRenderer: (row) => (
            <CodeHighlighter
              language="text"
              className="whitespace-pre-wrap break-words text-sm leading-relaxed"
              code={row.expression}
              copy={false}
              maxHeight="10rem"
              minWidth="20rem"
            />
          ),
        },
      ]}
    />
  );
}

function IdempotencySettings({
  idempotency,
}: {
  idempotency: NonNullable<WorkflowVersion['idempotency']>;
}) {
  return (
    <div className="space-y-3">
      <FieldGroup label="Expression">
        <CodeHighlighter
          language="text"
          className="whitespace-pre-wrap break-words text-sm leading-relaxed"
          code={idempotency.expression}
          copy={false}
          minWidth="20rem"
        />
      </FieldGroup>
      <div className="flex items-center gap-2">
        <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">
          TTL:
        </Label>
        <Badge variant="secondary" className="font-mono text-sm">
          {idempotency.ttlMs.toLocaleString()} ms
        </Badge>
      </div>
    </div>
  );
}

function ConfigurationSettings({ workflow }: { workflow: WorkflowVersion }) {
  const rows: { label: string; value: React.ReactNode }[] = [];

  if (workflow.sticky) {
    rows.push({ label: 'Sticky Strategy', value: workflow.sticky });
  }

  if (workflow.defaultPriority) {
    rows.push({ label: 'Default Priority', value: workflow.defaultPriority });
  }

  if (workflow.scheduleTimeout) {
    rows.push({ label: 'Schedule Timeout', value: workflow.scheduleTimeout });
  }

  if (rows.length === 0) {
    return (
      <EmptyState message="No additional configuration set for this workflow." />
    );
  }

  return (
    <div className="space-y-2">
      {rows.map((row) => (
        <div key={row.label} className="flex items-center gap-2">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">
            {row.label}:
          </Label>
          <Badge variant="secondary" className="font-mono text-sm">
            {row.value}
          </Badge>
        </div>
      ))}
    </div>
  );
}
