import { Badge } from '@/components/v1/ui/badge';
import { cn } from '@/lib/utils';

export function formatTenantRegionDisplay(
  region: string | undefined,
): string | undefined {
  if (!region) {
    return undefined;
  }
  const idx = region.indexOf(':');
  if (idx >= 0 && idx < region.length - 1) {
    return region.slice(idx + 1);
  }
  return region;
}

export function TenantRegionBadge({
  region,
  className,
}: {
  region: string | undefined;
  className?: string;
}) {
  const label = formatTenantRegionDisplay(region);
  if (!label) {
    return null;
  }

  return (
    <Badge
      variant="outline"
      className={cn(
        'max-w-[9rem] shrink-0 truncate px-1.5 py-0 text-[10px] font-normal',
        className,
      )}
      title={region}
    >
      {label}
    </Badge>
  );
}
