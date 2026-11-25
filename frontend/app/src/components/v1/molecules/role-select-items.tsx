import { SelectItem } from '@/components/v1/ui/select';
import { useTenantRoles } from '@/hooks/use-tenant-roles';

// Granting OWNER is limited by who is asking, which the role list cannot express, so callers decide it.
export function RoleSelectItems({ allowOwner }: { allowOwner: boolean }) {
  const { roles } = useTenantRoles();

  return (
    <>
      {roles
        .filter((role) => allowOwner || role.name !== 'OWNER')
        .map((role) => (
          <SelectItem key={role.name} value={role.name}>
            {role.label}
          </SelectItem>
        ))}
    </>
  );
}
