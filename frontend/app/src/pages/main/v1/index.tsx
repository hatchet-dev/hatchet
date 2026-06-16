import { SideNav } from '../../../components/v1/nav/side-nav';
import { sideNavItems } from './side-nav-items';
import { ThreeColumnLayout } from '@/components/layout/three-column-layout';
import { SidePanel } from '@/components/v1/nav/side-panel';
import { Loading } from '@/components/v1/ui/loading';
import useCloud from '@/hooks/use-cloud';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import {
  MembershipsContextType,
  UserContextType,
  useContextFromParent,
} from '@/lib/outlet';
import { OutletWithContext, useOutletContext } from '@/lib/router-helpers';
import { useQueryClient } from '@tanstack/react-query';
import { ReactNode, useEffect, useMemo, useRef } from 'react';

// How long (ms) prefetched data is considered fresh.
// Within this window, hovering or revisiting a route will NOT trigger a new API call.
const PREFETCH_STALE_TIME = 30_000; // 30 seconds

export function MainShell({ children }: { children?: ReactNode }) {
  const ctx = useOutletContext<UserContextType & MembershipsContextType>();
  const { user, memberships } = ctx;
  const { tenantId, isUserUniverseLoaded } = useTenantDetails();
  const { cloud, featureFlags, isCloudEnabled } = useCloud(tenantId);
  const managedWorkerEnabled = featureFlags?.['managed-worker'] === 'true';
  const { getOrganizationIdForTenant } = useOrganizations();
  const queryClient = useQueryClient();
  const orgId = isCloudEnabled
    ? tenantId
      ? (getOrganizationIdForTenant(tenantId) ?? undefined)
      : undefined
    : undefined;

  // FIX (Level 3): Track the last tenantId we prefetched for.
  // Without this, the effect re-runs and fires a fresh API request every time
  // the component re-renders with the same tenantId (e.g. on hover-triggered
  // re-renders caused by TanStack Router's preload="intent" on sidebar Links).
  const lastPrefetchedTenantId = useRef<string | null>(null);

  useEffect(() => {
    if (!tenantId) {
      return;
    }

    // FIX (Level 3): Only prefetch if we haven't already prefetched for this tenant,
    // OR if the query cache itself says the data is stale (older than PREFETCH_STALE_TIME).
    // This prevents redundant network calls when the user hovers across nav items
    // or when the component re-mounts without the tenantId actually changing.
    const queryState = queryClient.getQueryState(
      queries.workflows.list(tenantId, { limit: 200 }).queryKey,
    );

    const isDataFresh =
      queryState?.dataUpdatedAt != null &&
      Date.now() - queryState.dataUpdatedAt < PREFETCH_STALE_TIME;

    if (lastPrefetchedTenantId.current === tenantId && isDataFresh) {
      // Data is still fresh for this tenant — skip the network call entirely.
      return;
    }

    lastPrefetchedTenantId.current = tenantId;

    void queryClient.prefetchQuery({
      ...queries.workflows.list(tenantId, { limit: 200 }),
      // FIX (Level 3): staleTime tells React Query: "if the cached data is younger
      // than 30s, do NOT go to the network — serve from cache instead."
      // Without this, every prefetchQuery call unconditionally hits the API.
      staleTime: PREFETCH_STALE_TIME,
    });
  }, [queryClient, tenantId]);

  const navSections = useMemo(
    () =>
      sideNavItems({
        canBill: cloud?.canBill,
        managedWorkerEnabled,
        isCloudEnabled,
        orgId,
      }),
    [cloud?.canBill, managedWorkerEnabled, isCloudEnabled, orgId],
  );

  const childCtx = useContextFromParent({
    user,
    memberships,
  });

  const content = !isUserUniverseLoaded ? (
    <Loading className="items-center justify-center" />
  ) : (
    (children ?? <OutletWithContext context={childCtx} />)
  );

  return (
    <ThreeColumnLayout
      // FIX (Level 1): SideNav receives noPreload so every <Link> inside it
      // renders with preload={false}. This stops TanStack Router from firing
      // route loaders on mouseenter — which was the root cause of #4205.
      // Navigation still works normally; only the speculative hover-prefetch
      // is disabled for sidebar items specifically.
      sidebar={<SideNav navItems={navSections} noPreload />}
      sidePanel={<SidePanel />}
      // mainClassName="overflow-auto"
      mainContainerType="inline-size"
    >
      <div className="shadow-[inset_1px_0_0_0_hsl(var(--border)/0.5),inset_0_1px_0_0_hsl(var(--border)/0.5)] md:rounded-tl-xl p-4 h-full dark:bg-[#050A23] bg-background overflow-y-auto [scrollbar-gutter:stable_both-edges]">
        {content}
      </div>
    </ThreeColumnLayout>
  );
}

function Main() {
  return <MainShell />;
}

export default Main;
