import { TenantRegionBadge } from '@/components/v1/molecules/nav-bar/tenant-region-badge';
import { TenantSwitcherPanel } from '@/components/v1/molecules/nav-bar/tenant-switcher-panel';
import { useRecentTenants } from '@/components/v1/molecules/nav-bar/use-recent-tenants';
import { useTenantSwitcherShortcut } from '@/components/v1/molecules/nav-bar/use-tenant-switcher-shortcut';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';
import { Tenant, TenantMember } from '@/lib/api';
import { globalEmitter } from '@/lib/global-emitter';
import { cn } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { appRoutes } from '@/router';
import { BuildingOffice2Icon, PlusIcon } from '@heroicons/react/24/outline';
import { CaretSortIcon } from '@radix-ui/react-icons';
import {
  PopoverTrigger,
  Popover,
  PopoverContent,
  PopoverPortal,
} from '@radix-ui/react-popover';
import { useNavigate, useRouter } from '@tanstack/react-router';
import React, { useMemo, useRef } from 'react';

interface TenantSwitcherProps {
  className?: string;
  memberships: TenantMember[];
}
export function TenantSwitcher({
  className,
  memberships,
}: TenantSwitcherProps) {
  const { meta } = useApiMeta();
  const {
    setTenant: setCurrTenant,
    isUserUniverseLoaded,
    tenant,
  } = useTenantDetails();
  const [open, setOpen] = React.useState(false);
  const { hasOrganizations } = useOrganizations();
  const router = useRouter();
  const navigate = useNavigate();
  const triggerRef = useRef<HTMLButtonElement>(null);
  const recentTenantIds = useRecentTenants(tenant?.metadata.id);
  useTenantSwitcherShortcut(triggerRef, () => setOpen(true));

  const tenants = useMemo(
    () =>
      memberships
        .map((membership) => membership.tenant)
        .filter((t): t is Tenant => !!t),
    [memberships],
  );

  if (!tenant) {
    return (
      <Button
        variant="outline"
        size="sm"
        aria-label="Loading tenant"
        className={cn(
          'min-w-0 max-w-[320px] justify-between gap-2 bg-muted/20 shadow-none',
          className,
        )}
        disabled
      >
        <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
          <BuildingOffice2Icon className="size-4 shrink-0 opacity-60" />
          <span className="min-w-0 flex-1 truncate text-muted-foreground">
            Loading tenant…
          </span>
        </div>
        <Spinner className="mr-0" />
      </Button>
    );
  }

  const footerActions =
    meta?.allowCreateTenant && !hasOrganizations
      ? [
          {
            id: 'new-tenant',
            label: 'New tenant',
            icon: <PlusIcon className="size-3.5" />,
            dataCy: 'new-tenant',
            onSelect: () => {
              globalEmitter.emit('create-new-tenant', {});
              setOpen(false);
            },
          },
        ]
      : [];

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          ref={triggerRef}
          variant="outline"
          size="sm"
          role="combobox"
          aria-expanded={open}
          aria-label="Select a tenant"
          className={cn(
            'min-w-0 max-w-[320px] justify-between gap-2 bg-muted/20 shadow-none hover:bg-muted/30',
            open && 'bg-muted/30',
            className,
          )}
          disabled={!isUserUniverseLoaded || memberships.length === 0}
        >
          <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
            <BuildingOffice2Icon className="size-4 shrink-0" />
            <span className="min-w-0 flex-1 truncate">{tenant.name}</span>
            <TenantRegionBadge region={tenant.region} />
          </div>
          {!isUserUniverseLoaded ? (
            <Spinner className="mr-0" />
          ) : (
            <CaretSortIcon className="size-4 shrink-0 opacity-50" />
          )}
        </Button>
      </PopoverTrigger>
      {/* Portal so the popover can render above the mobile sidebar overlay (header is z-50). */}
      <PopoverPortal>
        <PopoverContent
          side="bottom"
          align="start"
          sideOffset={8}
          // Must render above the mobile sidebar overlay (`side-nav` uses z-[100]).
          className="z-[200] w-[420px] max-w-[calc(100vw-2rem)] overflow-hidden rounded-xl border bg-popover p-0 text-popover-foreground shadow-lg"
        >
          <TenantSwitcherPanel
            tenants={tenants}
            currentTenant={tenant}
            recentTenantIds={recentTenantIds}
            footerActions={footerActions}
            getTenantHref={(t) =>
              router.buildLocation({
                to: appRoutes.tenantRunsRoute.to,
                params: { tenant: t.metadata.id },
              }).href
            }
            onSelectTenant={(t) => {
              setCurrTenant(t);
              setOpen(false);
            }}
            onTenantSettings={(t) => {
              setOpen(false);
              setCurrTenant(t, { navigate: false });
              navigate({
                to: appRoutes.tenantSettingsOverviewRoute.to,
                params: { tenant: t.metadata.id },
              });
            }}
          />
        </PopoverContent>
      </PopoverPortal>
    </Popover>
  );
}
