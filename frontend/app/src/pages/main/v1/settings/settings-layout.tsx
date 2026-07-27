import { settingsNavGroups } from './settings-nav-items';
import { Button } from '@/components/v1/ui/button';
import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';
import {
  getOptionalStringParam,
  OutletWithContext,
} from '@/lib/router-helpers';
import { cn } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { Link, useMatchRoute, useParams } from '@tanstack/react-router';

export default function SettingsLayout() {
  const { tenantId, organizationId } = useTenantDetails();
  const params = useParams({ strict: false });
  const { cloud, isCloudEnabled } = useCloud(tenantId);
  const { isControlPlaneEnabled } = useControlPlane();
  const { organizations } = useOrganizations();
  const { meta } = useApiMeta();
  const matchRoute = useMatchRoute();

  // On /organizations/* routes, prefer the org from the URL over the one
  // resolved from the active tenant, which can belong to a different org
  const orgId =
    getOptionalStringParam(params, 'organization') ?? organizationId;

  const isOrganizationOwner =
    organizations.find((o) => o.metadata.id === orgId)?.isOwner ?? false;
  const canManageSso =
    isOrganizationOwner && (meta?.auth?.schemes || []).includes('sso');

  const groups = settingsNavGroups({
    tenantId,
    orgId,
    canBill: cloud?.canBill,
    isCloudEnabled,
    isControlPlaneEnabled,
    isOrganizationOwner,
    canManageSso,
  });

  return (
    <div className="flex h-full min-w-0 gap-x-6">
      <aside className="sticky top-0 w-56 shrink-0 self-start py-4">
        <h2 className="mb-4 px-2 text-lg font-semibold">Settings</h2>
        <div className="flex flex-col gap-y-6">
          {groups.map((group) => (
            <div key={group.key}>
              <h3 className="mb-2 px-2 text-xs font-mono tracking-widest uppercase text-muted-foreground">
                {group.title}
              </h3>
              <div className="flex flex-col gap-y-1">
                {group.items.map((item) => {
                  const selected = Boolean(
                    matchRoute({
                      to: item.to,
                      params: item.params,
                      fuzzy: !item.exact,
                    }),
                  );

                  return (
                    <Link
                      key={item.key}
                      to={item.to}
                      params={item.params}
                      preload={false}
                    >
                      <Button
                        variant="ghost"
                        className={cn(
                          'h-8 w-full justify-start pl-2 min-w-0 overflow-hidden',
                          selected && 'bg-slate-200 dark:bg-slate-800',
                        )}
                      >
                        <span className="truncate">{item.name}</span>
                      </Button>
                    </Link>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      </aside>
      <div className="min-w-0 flex-1">
        <OutletWithContext />
      </div>
    </div>
  );
}
