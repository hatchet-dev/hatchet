import { WorkflowTag } from '@/lib/api';
import { cn } from '@/lib/utils';

export function WorkflowTags({ tags }: { tags: WorkflowTag[] }) {
  return tags?.map((tag) => {
    return (
      <div
        key={tag.name}
        className={cn(
          `text-[${tag.color}] bg-[${tag.color}]/10 ring-[${tag.color}]/30`,
          'flex-none rounded-full px-2 py-1 text-xs font-medium ring-1 ring-inset',
        )}
      >
        {tag.name}
      </div>
    );
  });
}
