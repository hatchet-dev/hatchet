import { TenantRole } from './api/generated/data-contracts';

export function roleAllows(
  roles: TenantRole[],
  role: string | undefined,
  operation: string,
): boolean {
  return (
    roles.find((r) => r.name === role)?.operations.includes(operation) ?? false
  );
}
