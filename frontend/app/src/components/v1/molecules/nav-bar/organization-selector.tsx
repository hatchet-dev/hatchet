import { Button } from "@/components/v1/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/v1/ui/command";
import { useAnalytics } from "@/hooks/use-analytics";
import { useTenantDetails } from "@/hooks/use-tenant";
import { OrganizationForUser } from "@/lib/api/generated/cloud/data-contracts";
import { cn } from "@/lib/utils";
import { useUserUniverse } from "@/providers/user-universe";
import { appRoutes } from "@/router";
import {
  BuildingOffice2Icon,
  CheckIcon,
  ChevronUpDownIcon,
  Cog6ToothIcon,
  PlusIcon,
} from "@heroicons/react/24/outline";
import {
  Popover,
  PopoverContent,
  PopoverPortal,
  PopoverTrigger,
} from "@radix-ui/react-popover";
import { Link, useNavigate } from "@tanstack/react-router";
import { useState, useMemo, useCallback } from "react";
import invariant from "tiny-invariant";

export function OrganizationSelector({ className }: { className?: string }) {
  const navigate = useNavigate();
  const {
    setTenant,
    isUserUniverseLoaded: isTenantLoaded,
    tenant,
  } = useTenantDetails();
  const {
    organizations,
    isLoaded: isUniverseLoaded,
    getOrganizationForTenant,
    getTenantWithTenantId,
  } = useUserUniverse();
  const { capture } = useAnalytics();
  const [open, setOpen] = useState(false);

  const currentOrg = useMemo(() => {
    if (!tenant || !getOrganizationForTenant) {
      return null;
    }
    return getOrganizationForTenant(tenant.metadata.id);
  }, [tenant, getOrganizationForTenant]);

  const sortedOrgs = useMemo(() => {
    if (!organizations) {
      return [];
    }
    return [...organizations].sort((a, b) => {
      if (a.isOwner !== b.isOwner) {
        return a.isOwner ? -1 : 1;
      }
      return a.name.localeCompare(b.name, undefined, {
        sensitivity: "base",
      });
    });
  }, [organizations]);

  const handleOrgSelect = useCallback(
    (org: OrganizationForUser) => {
      invariant(isUniverseLoaded);
      const firstTenant = org.tenants.at(0);
      invariant(firstTenant);

      capture("organization_selector_clicked", {
        organization_id: org.metadata.id,
      });

      setTenant(getTenantWithTenantId(firstTenant.id));

      setOpen(false);
    },
    [isUniverseLoaded, getTenantWithTenantId, setTenant, capture],
  );

  const handleSettingsClick = useCallback(
    (e: React.MouseEvent, org: OrganizationForUser) => {
      e.preventDefault();
      e.stopPropagation();

      setOpen(false);
      navigate({
        to: appRoutes.organizationsRoute.to,
        params: { organization: org.metadata.id },
      });
    },
    [navigate],
  );

  const triggerDisabled =
    !isTenantLoaded || !isUniverseLoaded || !organizations?.length;

  return (
    <Popover
      open={open}
      onOpenChange={(open) => {
        if (open) {
          capture("organization_selector_opened");
        }
        setOpen(open);
      }}
    >
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          role="combobox"
          aria-expanded={open}
          aria-label="Select an organization"
          className={cn(
            "w-[150px] md:w-[200px] justify-between gap-2 bg-muted/20 shadow-none hover:bg-muted/30",
            open && "bg-muted/30",
            className,
          )}
          disabled={triggerDisabled}
        >
          <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
            <BuildingOffice2Icon className="size-4 shrink-0" />
            <span className="min-w-0 flex-1 truncate">
              {currentOrg?.name ?? "Loading…"}
            </span>
          </div>
          {(!isTenantLoaded || !isUniverseLoaded) && !open ? (
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-muted-foreground/70" />
          ) : (
            <ChevronUpDownIcon className="size-4 shrink-0 opacity-50" />
          )}
        </Button>
      </PopoverTrigger>
      <PopoverPortal>
        <PopoverContent
          side="bottom"
          align="start"
          sideOffset={8}
          className="z-[200] w-[287px] rounded-md border border-border p-0 shadow-md"
        >
          <Command className="border-0">
            <CommandList>
              <CommandEmpty>No organizations found.</CommandEmpty>
              <CommandGroup>
                {sortedOrgs.map((org) => (
                  <CommandItem
                    key={org.metadata.id}
                    value={`org-${org.metadata.id}`}
                    onSelect={() => handleOrgSelect(org)}
                    className="cursor-pointer text-sm hover:bg-accent focus:bg-accent"
                  >
                    <div className="flex w-full items-center justify-between">
                      <span className="min-w-0 flex-1 truncate">
                        {org.name}
                      </span>
                      <div className="ml-2 flex flex-shrink-0 items-center gap-1">
                        <CheckIcon
                          className={cn(
                            "size-4",
                            currentOrg?.metadata.id === org.metadata.id
                              ? "opacity-100"
                              : "opacity-0",
                          )}
                        />
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-5 w-5 p-0 hover:bg-accent-foreground/10"
                          onClick={(e) => handleSettingsClick(e, org)}
                          title="Settings"
                        >
                          <Cog6ToothIcon className="size-3" />
                        </Button>
                      </div>
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
              <div className="px-2 py-1">
                <Button
                  variant="outline"
                  size="sm"
                  fullWidth
                  leftIcon={<PlusIcon className="size-4" />}
                  asChild
                >
                  <Link
                    to={appRoutes.organizationsNewRoute.to}
                    onClick={() => setOpen(false)}
                  >
                    Create Organization
                  </Link>
                </Button>
              </div>
            </CommandList>
          </Command>
        </PopoverContent>
      </PopoverPortal>
    </Popover>
  );
}
