import { Badge } from '@/next/components/ui/badge';

export const WorkerActions = ({ actions }: { actions: string[] }) => {
  return (
    <div className="flex flex-col gap-y-2">
      <p className="text-lg">Actions</p>
      <div className="gap-2">
        {actions?.map((action) => (
          <Badge
            key={action}
            variant="outline"
            className="h-8 p-2 m-1 text-base font-normal hover:text-purple-300"
          >
            {action}
          </Badge>
        ))}
      </div>
    </div>
  );
};
