import { Button } from '@/components/v1/ui/button';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';

interface ParentTaskBannerProps {
  parentTaskExternalId?: string;
  onClear: () => void;
}

export const ParentTaskBanner = ({
  parentTaskExternalId,
  onClear,
}: ParentTaskBannerProps) => {
  const parentTaskRun = useQuery({
    ...queries.v1Tasks.get(parentTaskExternalId || ''),
    enabled: !!parentTaskExternalId,
  });

  if (!parentTaskExternalId || parentTaskRun.isLoading || !parentTaskRun.data) {
    return null;
  }

  return (
    <div className="flex flex-row items-center gap-x-2">
      <p>Child runs of parent:</p>
      <p className="font-semibold text-orange-300">
        {parentTaskRun.data.displayName}
      </p>
      <Button variant="outline" className="ml-4" onClick={onClear}>
        Clear
      </Button>
    </div>
  );
};
