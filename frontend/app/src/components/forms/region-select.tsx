import { usePylon } from '@/components/support-chat';
import { formatTenantRegionDisplay } from '@/components/v1/molecules/nav-bar/tenant-region-badge';
import { Button } from '@/components/v1/ui/button';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { OrganizationAvailableShard } from '@/lib/api/generated/control-plane/data-contracts';

const OFFICE_HOURS_URL = 'https://hatchet.run/office-hours';

export function shardDeploymentKey(shard: OrganizationAvailableShard): string {
  if (shard.provider) {
    return `${shard.provider}:${shard.region}`;
  }
  return shard.region;
}

type RegionSelectProps = {
  shards: OrganizationAvailableShard[];
  value: string | undefined;
  onValueChange: (regionKey: string) => void;
  isLoading: boolean;
  disabled?: boolean;
  id?: string;
};

export function RegionSelect({
  shards,
  value,
  onValueChange,
  isLoading,
  disabled = false,
  id = 'deployment-region',
}: RegionSelectProps) {
  const pylon = usePylon();
  const selectDisabled = disabled || isLoading || shards.length <= 1;

  return (
    <div className="grid gap-2">
      <Label htmlFor={id}>Region</Label>
      <p className="text-sm text-muted-foreground">
        Choose where this tenant&apos;s control plane and data are deployed.
      </p>
      <Select
        value={value}
        onValueChange={onValueChange}
        disabled={selectDisabled}
      >
        <SelectTrigger id={id}>
          <SelectValue
            placeholder={isLoading ? 'Loading regions…' : 'Select a region'}
          />
        </SelectTrigger>
        <SelectContent>
          {shards.map((shard) => {
            const key = shardDeploymentKey(shard);
            const label = formatTenantRegionDisplay(key) ?? key;
            return (
              <SelectItem key={key} value={key}>
                {label}
              </SelectItem>
            );
          })}
        </SelectContent>
      </Select>
      <p className="text-sm text-muted-foreground">
        Don&apos;t see your region? Reach out via{' '}
        {pylon.enabled ? (
          <>
            <Button
              type="button"
              variant="link"
              className="h-auto p-0 text-sm font-normal"
              onClick={() => pylon.show()}
            >
              Open support chat
            </Button>
            , or{' '}
          </>
        ) : null}
        <a
          href={OFFICE_HOURS_URL}
          target="_blank"
          rel="noopener noreferrer"
          className="text-primary underline-offset-4 hover:underline"
        >
          Schedule office hours
        </a>
        .
      </p>
    </div>
  );
}
