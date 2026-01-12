import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { CopyWorkflowConfigButton } from '@/components/v1/shared/copy-workflow-config';
import { Badge } from '@/components/v1/ui/badge';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Label } from '@/components/v1/ui/label';
import {
  ConcurrencyLimitStrategy,
  ConcurrencyScope,
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
  const hasTriggers =
    (workflow.triggers?.events && workflow.triggers.events.length > 0) ||
    (workflow.triggers?.crons && workflow.triggers.crons.length > 0);

  return (
    <div className="space-y-5">
      {hasTriggers && (
        <SettingsSection title="Triggers">
          <TriggerSettings workflow={workflow} />
        </SettingsSection>
      )}

      <SettingsSection title="Concurrency">
        <ConcurrencySettings workflow={workflow} />
      </SettingsSection>

      <SettingsSection title="Other">
        <ConfigurationSettings workflow={workflow} />
      </SettingsSection>

      <CopyWorkflowConfigButton workflowConfig={workflow.workflowConfig} />
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
  description,
}: {
  label: string;
  children: React.ReactNode;
  description?: string;
}) {
  return (
    <div className="space-y-1">
      <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </Label>
      {children}
      {description && (
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {description}
        </p>
      )}
    </div>
  );
}

// function ScheduleTimeoutSettings({ workflow }: { workflow: WorkflowVersion }) {
//   if (!workflow.scheduleTimeout) {
//     return (
//       <div className="text-[0.8rem] text-gray-700 dark:text-gray-300">
//         No schedule timeout set for this workflow.
//       </div>
//     );
//   }

//   return (
//     <div className="flex flex-col gap-2">
//       <Input
//         disabled
//         placeholder="Schedule Timeout"
//         value={workflow.scheduleTimeout}
//       />
//     </div>
//   );
// }

function TriggerSettings({ workflow }: { workflow: WorkflowVersion }) {
  if (!workflow.triggers) {
    return (
      <EmptyState message="There are no trigger settings for this workflow." />
    );
  }

  return (
    <div className="space-y-2">
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
                <Badge
                  key={cronTrigger.cron}
                  variant="secondary"
                  className="font-mono text-sm"
                >
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
          // hack for typing
          metadata: {
            id: '',
          },
        }))
        .sort(
          (a, b) =>
            b.scope.localeCompare(a.scope) ||
            a.stepReadableId.localeCompare(b.stepReadableId),
        )}
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

function ConfigurationSettings({ workflow }: { workflow: WorkflowVersion }) {
  const hasConfig = workflow.sticky || workflow.defaultPriority;

  if (!hasConfig) {
    return (
      <EmptyState message="No additional configuration set for this workflow." />
    );
  }

  return (
    <div className="space-y-2">
      {workflow.sticky && (
        <div className="flex items-center gap-2">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">
            Sticky Strategy:
          </Label>
          <Badge variant="secondary" className="font-mono text-sm">
            {workflow.sticky}
          </Badge>
        </div>
      )}

      {workflow.defaultPriority && (
        <div className="flex items-center gap-2">
          <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">
            Default Priority:
          </Label>
          <Badge variant="secondary" className="font-mono text-sm">
            {workflow.defaultPriority}
          </Badge>
        </div>
      )}
    </div>
  );
}
