import { TenantSwitcher } from './tenant-switcher';
import { useOrganizations } from '@/hooks/use-organizations';

export function TenantBreadcrumbs() {
  const { enabled: organizationsEnabled } = useOrganizations();
  if (!organizationsEnabled) {
    return null;
  }
  return (
    <div className="min-w-0 px-8">
      / yo <TenantSwitcher /> / yo <TenantSwitcher />
    </div>
  );
}
