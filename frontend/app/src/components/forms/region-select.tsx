import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { OrganizationAvailableShard } from '@/lib/api/generated/control-plane/data-contracts';
import {
  formatShardDeploymentKey,
  shardDeploymentKey,
} from '@/lib/shard-deployment-key';

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
  const selectDisabled = disabled || isLoading || shards.length <= 1;

  return (
    <div className="grid gap-2">
      <Label htmlFor={id}>Region</Label>
      <p className="text-sm text-muted-foreground">
        Choose where your first tenant is deployed.
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
            const label = formatShardDeploymentKey(key) ?? shard.region;
            return (
              <SelectItem key={key} value={key}>
                {label}
              </SelectItem>
            );
          })}
        </SelectContent>
      </Select>
    </div>
  );
}
