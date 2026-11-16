import { Trigger, TriggerBadge, triggerKey } from './trigger-badge';
import { TriggersDialog } from './triggers-dialog';
import { Workflow } from '@/lib/api';

const MAX_VISIBLE_TRIGGERS = 2;

export function WorkflowTriggers({ workflow }: { workflow: Workflow }) {
  const triggers: Trigger[] = [
    ...(workflow.triggers?.events ?? []).flatMap((event) =>
      event.event_key
        ? [{ type: 'event' as const, eventKey: event.event_key }]
        : [],
    ),
    ...(workflow.triggers?.crons ?? []).flatMap((cron) =>
      cron.cron ? [{ type: 'cron' as const, cron: cron.cron }] : [],
    ),
  ];

  if (triggers.length === 0) {
    return <span className="text-muted-foreground">—</span>;
  }

  const hidden = triggers.length - MAX_VISIBLE_TRIGGERS;

  return (
    <div className="flex flex-col items-start gap-1">
      {triggers.slice(0, MAX_VISIBLE_TRIGGERS).map((trigger) => (
        <TriggerBadge key={triggerKey(trigger)} trigger={trigger} />
      ))}
      {hidden > 0 && (
        <TriggersDialog workflowName={workflow.name} triggers={triggers}>
          <button
            type="button"
            className="text-xs text-muted-foreground hover:text-foreground hover:underline"
          >
            +{hidden} more
          </button>
        </TriggersDialog>
      )}
    </div>
  );
}
