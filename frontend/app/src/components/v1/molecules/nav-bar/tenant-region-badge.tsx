import { Badge } from '@/components/v1/ui/badge';
import { formatShardDeploymentKey } from '@/lib/shard-deployment-key';
import { cn } from '@/lib/utils';

export function TenantRegionBadge({
  region,
  className,
}: {
  region: string | undefined;
  className?: string;
}) {
  const label = formatShardDeploymentKey(region);
  if (!label) {
    return null;
  }

  return (
    <Badge
      variant="outline"
      className={cn(
        'max-w-[9rem] shrink-0 px-1.5 py-0 text-[10px] font-normal',
        className,
      )}
      title={region}
    >
      {/* Badge is a flex container, so the ellipsis needs its own element. */}
      <span className="truncate">{label}</span>
    </Badge>
  );
}
