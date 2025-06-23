import { Button } from '@/components/ui/button';
import { Badge } from '@/components/v1/ui/badge';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { WorkflowVersion } from '@/lib/api';
import CronPrettifier from 'cronstrue';
import { CheckIcon, CopyIcon } from 'lucide-react';
import { useState } from 'react';

export default function WorkflowGeneralSettings({
  workflow,
}: {
  workflow: WorkflowVersion;
}) {
  const [copySuccess, setCopySuccess] = useState(false);

  return (
    <div className="space-y-5">
      <SettingsSection title="Triggers">
        <TriggerSettings workflow={workflow} />
      </SettingsSection>

      <SettingsSection title="Concurrency">
        <ConcurrencySettings workflow={workflow} />
      </SettingsSection>

      <SettingsSection title="Other">
        <ConfigurationSettings workflow={workflow} />
      </SettingsSection>

      <Button
        variant="outline"
        size="sm"
        onClick={() => {
          navigator.clipboard.writeText(
            JSON.stringify(workflow.workflowConfig),
          );
          setCopySuccess(true);
          setTimeout(() => setCopySuccess(false), 2000);
        }}
        className="mt-6"
      >
        {copySuccess ? (
          <>
            <CheckIcon className="w-3 h-3 mr-2" />
            Copied!
          </>
        ) : (
          <>
            <CopyIcon className="w-3 h-3 mr-2" />
            Copy Config
          </>
        )}
      </Button>
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
    <div className="space-y-2">
      <h3 className="text-base font-semibold text-gray-900 dark:text-gray-100">
        {title}
      </h3>
      {children}
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <p className="text-sm text-gray-500 dark:text-gray-400 italic">{message}</p>
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
                <Input
                  disabled
                  value={cronTrigger.cron}
                  className="font-mono h-8"
                />
                {cronTrigger.cron && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Runs{' '}
                    {CronPrettifier.toString(cronTrigger.cron).toLowerCase()}{' '}
                    UTC
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
  if (!workflow.concurrency) {
    return (
      <EmptyState message="There are no concurrency settings for this workflow." />
    );
  }

  return (
    <div className="space-y-2">
      <FieldGroup
        label="Max Runs"
        description="Maximum number of concurrent workflow runs"
      >
        <Input
          disabled
          value={workflow.concurrency.maxRuns}
          className="font-mono h-8"
        />
      </FieldGroup>

      <FieldGroup
        label="Strategy"
        description="What happens when max runs is reached"
      >
        <Input
          disabled
          value={workflow.concurrency.limitStrategy}
          className="font-mono h-8"
        />
      </FieldGroup>
    </div>
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
