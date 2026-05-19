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
import { ReactNode, useEffect, useMemo } from 'react';

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

  useEffect(() => {
    if (!tenantId) {
      return;
    }

    void queryClient.prefetchQuery(
      queries.workflows.list(tenantId, { limit: 200 }),
    );
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
      sidebar={<SideNav navItems={navSections} />}
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
