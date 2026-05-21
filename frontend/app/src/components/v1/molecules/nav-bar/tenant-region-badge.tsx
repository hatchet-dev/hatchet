import { Badge } from '@/components/v1/ui/badge';
import { OrganizationAvailableShard } from '@/lib/api/generated/control-plane/data-contracts';
import { cn } from '@/lib/utils';

export function shardDeploymentKey(shard: OrganizationAvailableShard): string {
  if (shard.provider) {
    return `${shard.provider}:${shard.region}`;
  }
  return shard.region;
}

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

export function formatTenantDeploymentBadgeLabel({
  region,
  shardDisplayName,
}: {
  region?: string;
  shardDisplayName?: string;
}): string | undefined {
  const trimmed = shardDisplayName?.trim();
  if (trimmed) {
    return trimmed;
  }
  return formatTenantRegionDisplay(region);
}

export function formatTenantDeploymentBadgeTooltip({
  region,
  shardDisplayName,
}: {
  region?: string;
  shardDisplayName?: string;
}): string | undefined {
  if (!region) {
    return shardDisplayName?.trim() || undefined;
  }
  const trimmed = shardDisplayName?.trim();
  if (trimmed) {
    return `${trimmed} (${region})`;
  }
  return region;
}

export function TenantRegionBadge({
  region,
  shardDisplayName,
  className,
}: {
  region?: string;
  shardDisplayName?: string;
  className?: string;
}) {
  const label = formatTenantDeploymentBadgeLabel({ region, shardDisplayName });
  if (!label) {
    return null;
  }

  const title = formatTenantDeploymentBadgeTooltip({
    region,
    shardDisplayName,
  });

  return (
    <Badge
      variant="outline"
      className={cn(
        'max-w-[9rem] shrink-0 truncate px-1.5 py-0 text-[10px] font-normal',
        className,
      )}
      title={title}
    >
      {label}
    </Badge>
  );
}
