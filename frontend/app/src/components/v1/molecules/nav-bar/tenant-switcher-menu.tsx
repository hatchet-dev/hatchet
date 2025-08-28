import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import { CheckIcon } from '@heroicons/react/24/outline';
import invariant from 'tiny-invariant';
import { TenantMember, TenantVersion } from '@/lib/api';
import { useCallback } from 'react';
import { useTenantDetails } from '@/hooks/use-tenant';

interface TenantSwitcherMenuProps {
  tenants: TenantMember[];
  organizationName: string;
  onTenantSwitch?: () => void;
}

export function TenantSwitcherMenu({
  tenants,
  organizationName,
  onTenantSwitch,
}: TenantSwitcherMenuProps) {
  const { setTenant: setCurrTenant, tenant: currTenant } = useTenantDetails();

  const handleTenantSelect = useCallback(
    (membership: TenantMember) => {
      invariant(membership.tenant);
      setCurrTenant(membership.tenant);
      onTenantSwitch?.();

      if (membership.tenant.version === TenantVersion.V0) {
        setTimeout(() => {
          window.location.href = `/workflow-runs?tenant=${membership.tenant?.metadata.id}`;
        }, 0);
      }
    },
    [setCurrTenant, onTenantSwitch],
  );

  return (
    <div className="min-w-[220px] bg-background border border-border rounded-md shadow-lg p-2">
      <div className="text-xs text-muted-foreground px-2 py-1 font-medium border-b border-border mb-2">
        {organizationName} Tenants ({tenants.length})
      </div>
      <div className="space-y-0.5">
        {tenants.map((membership) => (
          <Button
            key={membership.metadata.id}
            variant="ghost"
            size="sm"
            onClick={() => handleTenantSelect(membership)}
            className="w-full justify-start h-8 px-2 text-sm font-normal"
          >
            <div className="w-2 h-2 rounded-full bg-primary mr-2 flex-shrink-0" />
            <span className="truncate">{membership.tenant?.name}</span>
            <CheckIcon
              className={cn(
                'ml-auto h-3 w-3 flex-shrink-0',
                currTenant?.slug === membership.tenant?.slug
                  ? 'opacity-100 text-primary'
                  : 'opacity-0',
              )}
            />
          </Button>
        ))}
      </div>
    </div>
  );
}
