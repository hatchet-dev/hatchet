import { OrganizationAvailableShard } from '@/lib/api/generated/control-plane/data-contracts';

/**
 * Builds the compact API selector used when creating tenants.
 * Unnamed shard rows render as `provider:region`; named shard rows render as `provider:region:shardName`.
 */
export function shardDeploymentKey(shard: OrganizationAvailableShard): string {
  if (shard.provider) {
    const base = `${shard.provider}:${shard.region}`;
    return shard.shardName ? `${base}:${shard.shardName}` : base;
  }
  return shard.shardName ? `${shard.region}:${shard.shardName}` : shard.region;
}

/**
 * Human-readable label for a compact deployment target or tenant `region` string.
 * Unnamed: `aws:us-west-2` -> `us-west-2`. Named: `aws:us-west-2:west` -> `us-west-2 / west`.
 */
export function formatShardDeploymentKey(
  region: string | undefined,
): string | undefined {
  if (!region) {
    return undefined;
  }

  const parts = region.split(':');
  if (parts.length >= 3) {
    const cloudRegion = parts[1];
    const shardName = parts.slice(2).join(':');
    return `${cloudRegion} / ${shardName}`;
  }
  if (parts.length === 2) {
    return parts[1];
  }
  return region;
}
