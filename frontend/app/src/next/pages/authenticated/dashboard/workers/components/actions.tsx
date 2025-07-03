import { Badge } from '@/next/components/ui/badge';

export const WorkerActions = ({ actions }: { actions: string[] }) => {
  return (
    <div className="flex flex-col gap-y-2">
      <p className="text-lg">Actions</p>
      <p className="text-sm text-muted-foreground">
        An action is a task that can be performed in a workflow. Different
        workers can register sets of actions.
      </p>
      <div className="gap-2">
        {actions?.map((action) => (
          <Badge key={action} variant="outline" className="p-2">
            {action}
          </Badge>
        ))}
      </div>
    </div>
  );
};
