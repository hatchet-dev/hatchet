import { CreateOrganizationDialog } from '@/components/v1/molecules/nav-bar/create-organization-dialog';
import { TenantSwitcherLoadingButton } from '@/components/v1/molecules/nav-bar/tenant-switcher-loading';
import { TenantSwitcherMenu } from '@/components/v1/molecules/nav-bar/tenant-switcher-menu';
import { TenantSwitcherTriggerButton } from '@/components/v1/molecules/nav-bar/tenant-switcher-trigger';
import { useTenantSwitcherGroups } from '@/components/v1/molecules/nav-bar/use-tenant-switcher-groups';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';
import { TenantMember } from '@/lib/api';
import { cn } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { appRoutes } from '@/router';
import {
  PopoverTrigger,
  Popover,
  PopoverContent,
  PopoverPortal,
} from '@radix-ui/react-popover';
import { useLocation, useNavigate } from '@tanstack/react-router';
import React from 'react';

interface TenantSwitcherProps {
  className?: string;
  memberships: TenantMember[];
  /**
   * When true, tenants will be grouped under organizations (if org data exists).
   * Intended for cloud mode.
   */
  enableOrganizations?: boolean;
  /** When true, the trigger button stretches to full width (used in some headers). */
  fullWidth?: boolean;
}
export function TenantSwitcher({
  className,
  memberships,
  enableOrganizations = true,
  fullWidth,
}: TenantSwitcherProps) {
  const { meta } = useApiMeta();
  const {
    setTenant: setCurrTenant,
    isLoading: isTenantLoading,
    tenant,
  } = useTenantDetails();
  const [open, setOpen] = React.useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const [showCreateOrgModal, setShowCreateOrgModal] = React.useState(false);
  const [orgName, setOrgName] = React.useState('');
  const {
    organizations,
    getOrganizationForTenant,
    isTenantArchivedInOrg,
    handleCreateOrganization,
    createOrganizationLoading,
    isLoading: isOrganizationsLoading,
    hasOrganizations,
  } = useOrganizations();

  const shouldGroupByOrganization =
    Boolean(enableOrganizations) && Boolean(hasOrganizations);

  const handleNavigate = (nav: {
    to: string;
    params?: Record<string, string>;
  }) => {
    if (!nav.to) {
      return;
    }

    // Store the current path before navigating to org settings
    sessionStorage.setItem('orgSettingsPreviousPath', location.pathname);
    navigate({ to: nav.to, params: nav.params, replace: false });
  };

  const { orgGroups, standaloneTenants } = useTenantSwitcherGroups({
    shouldGroupByOrganization,
    memberships,
    organizations,
    getOrganizationForTenant,
    isTenantArchivedInOrg,
  });

  const triggerDisabled =
    isTenantLoading ||
    (shouldGroupByOrganization && isOrganizationsLoading) ||
    memberships.length === 0;

  if (!tenant) {
    return (
      <TenantSwitcherLoadingButton
        className={className}
        fullWidth={fullWidth}
      />
    );
  }

  return (
    <>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <TenantSwitcherTriggerButton
            className={className}
            fullWidth={fullWidth}
            open={open}
            tenantName={tenant.name}
            tenantColor={tenant.color}
            shouldGroupByOrganization={shouldGroupByOrganization}
            disabled={triggerDisabled}
            showSpinner={
              (isTenantLoading ||
                (shouldGroupByOrganization && isOrganizationsLoading)) &&
              !open
            }
          />
        </PopoverTrigger>
        {/* Portal so the popover can render above the mobile sidebar overlay (header is z-50). */}
        <PopoverPortal>
          <PopoverContent
            side="bottom"
            align="start"
            sideOffset={8}
            // Must render above the mobile sidebar overlay (`side-nav` uses z-[100]).
            className={cn(
              'z-[300] p-0',
              shouldGroupByOrganization
                ? 'w-[680px] max-w-[calc(100vw-2rem)]'
                : 'w-56',
            )}
          >
            <TenantSwitcherMenu
              shouldGroupByOrganization={shouldGroupByOrganization}
              memberships={memberships}
              tenantSlug={tenant.slug}
              orgGroups={orgGroups}
              standaloneTenants={standaloneTenants}
              onSelectTenant={(t) => setCurrTenant(t)}
              onClose={() => setOpen(false)}
              onManageOrganizations={() =>
                handleNavigate({ to: appRoutes.organizationsRoute.to })
              }
              onCreateOrganizationClick={() => setShowCreateOrgModal(true)}
              allowCreateTenant={Boolean(meta?.allowCreateTenant)}
              hasOrganizations={hasOrganizations}
            />
          </PopoverContent>
        </PopoverPortal>
      </Popover>

      {shouldGroupByOrganization && (
        <CreateOrganizationDialog
          open={showCreateOrgModal}
          onOpenChange={setShowCreateOrgModal}
          orgName={orgName}
          setOrgName={setOrgName}
          createOrganizationLoading={createOrganizationLoading}
          onCreate={(name) => {
            handleCreateOrganization(name, () => {
              setShowCreateOrgModal(false);
              setOrgName('');
            });
          }}
        />
      )}
    </>
  );
}
