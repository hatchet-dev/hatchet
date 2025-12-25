import { TenantColorDot } from '../molecules/nav-bar/tenant-color-dot';
import { TenantSwitcher } from './tenant-switcher';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/v1/ui/breadcrumb';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';

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
        {activeOrganization && (
          <>
            <BreadcrumbItem className="min-w-0">
              <span className="group flex min-w-0 items-center gap-2">
                <span
                  className="min-w-0 truncate"
                  title={activeOrganization.name}
                >
                  {activeOrganization.name}
                </span>
                <TenantSwitcher
                  tone="chromeless"
                  className="shrink-0 group-hover:bg-muted/30"
                />
              </span>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="shrink-0 opacity-60" />
          </>
        )}

        {tenant && (
          <BreadcrumbItem className="min-w-0">
            <BreadcrumbPage className="min-w-0">
              <span className="group flex min-w-0 items-center gap-2">
                <TenantColorDot color={tenant?.color} size="md" />
                <span className="min-w-0 truncate" title={tenant?.name}>
                  {tenant?.name ?? 'Loading tenant…'}
                </span>
                <TenantSwitcher
                  tone="chromeless"
                  className="shrink-0 group-hover:bg-muted/30"
                />
              </span>
            </BreadcrumbPage>
          </BreadcrumbItem>
        )}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
