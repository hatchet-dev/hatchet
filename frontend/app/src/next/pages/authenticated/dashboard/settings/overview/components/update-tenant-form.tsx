import { useState, useEffect } from 'react';
import { Button } from '@/next/components/ui/button';
import { Input } from '@/next/components/ui/input';
import { Label } from '@/next/components/ui/label';
import {
  Tenant,
  UpdateTenantRequest,
} from '@/lib/api/generated/data-contracts';

interface UpdateTenantFormProps {
  tenant: Tenant;
  isLoading: boolean;
  onSubmit: (data: UpdateTenantRequest) => void;
}

export function UpdateTenantForm({
  tenant,
  isLoading,
  onSubmit,
}: UpdateTenantFormProps) {
  const [name, setName] = useState(tenant.name);
  const [changed, setChanged] = useState(false);

  useEffect(() => {
    setName(tenant.name);
    setChanged(false);
  }, [tenant]);

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setName(e.target.value);
    setChanged(e.target.value !== tenant.name);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({ name });
    setChanged(false);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4 w-full max-w-md">
      <div className="space-y-2">
        <Label htmlFor="tenant-name">Tenant Name</Label>
        <Input
          id="tenant-name"
          value={name}
          onChange={handleNameChange}
          placeholder="Enter tenant name"
        />
      </div>

      {changed && (
        <Button type="submit" loading={isLoading || !name.trim()}>
          Save Changes
        </Button>
      )}
    </form>
  );
}
