import { TenantColorDot } from '../molecules/nav-bar/tenant-color-dot';
import { TenantSwitcher } from './tenant-switcher';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbLink,
  BreadcrumbSeparator,
} from '@/components/v1/ui/breadcrumb';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';
import { appRoutes } from '@/router';
import { Link } from '@tanstack/react-router';

export function TenantBreadcrumbs() {
  const { tenant } = useTenantDetails();
  const { enabled: organizationsEnabled, activeOrganization } =
    useOrganizations();
  if (!organizationsEnabled) {
    return null;
  }

  // Organizations are enabled, but the active org/tenant may not be loaded yet.
  // Avoid rendering an empty breadcrumb row.
  if (!activeOrganization && !tenant) {
    return null;
  }

  return (
    <Breadcrumb className="min-w-0">
      <BreadcrumbList className="min-w-0 flex-nowrap overflow-hidden">
        {activeOrganization?.metadata?.id && (
          <>
            <BreadcrumbItem className="min-w-0">
              <span className="group flex min-w-0 items-center gap-2">
                <BreadcrumbLink asChild className="min-w-0 truncate">
                  <Link
                    to={appRoutes.organizationsRoute.to}
                    params={{ organization: activeOrganization.metadata.id }}
                    title={activeOrganization.name}
                  >
                    {activeOrganization.name}
                  </Link>
                </BreadcrumbLink>
                <TenantSwitcher
                  tone="chromeless"
                  className="shrink-0 group-hover:bg-muted/30"
                />
              </span>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="shrink-0 opacity-60" />
          </>
        )}

        {tenant?.metadata?.id && (
          <BreadcrumbItem className="min-w-0">
            <span className="group flex min-w-0 items-center gap-2">
              <TenantColorDot color={tenant.color} size="md" />
              <BreadcrumbLink asChild className="min-w-0 truncate">
                <Link
                  to={appRoutes.tenantRoute.to}
                  params={{ tenant: tenant.metadata.id }}
                  title={tenant.name}
                >
                  {tenant.name}
                </Link>
              </BreadcrumbLink>
              <TenantSwitcher
                tone="chromeless"
                className="shrink-0 group-hover:bg-muted/30"
              />
            </span>
          </BreadcrumbItem>
        )}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
