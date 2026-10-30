import {
  Trigger,
  TriggerBadge,
  triggerDescription,
  triggerKey,
} from './trigger-badge';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/v1/ui/dialog';

export function TriggersDialog({
  workflowName,
  triggers,
  children,
}: {
  workflowName: string;
  triggers: Trigger[];
  children: React.ReactNode;
}) {
  return (
    <Dialog>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="w-full max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Triggers for {workflowName}</DialogTitle>
        </DialogHeader>
        <div className="max-h-96 space-y-2 overflow-y-auto">
          {triggers.map((trigger) => (
            <div key={triggerKey(trigger)} className="flex items-center gap-2">
              <TriggerBadge trigger={trigger} />
              <span className="text-xs text-muted-foreground">
                {triggerDescription(trigger)}
              </span>
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
