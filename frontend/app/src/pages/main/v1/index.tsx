import { SideNav } from '../../../components/v1/nav/side-nav';
import { sideNavItems } from './side-nav-items';
import { ThreeColumnLayout } from '@/components/layout/three-column-layout';
import { SidePanel } from '@/components/v1/nav/side-panel';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import {
  MembershipsContextType,
  UserContextType,
  useContextFromParent,
} from '@/lib/outlet';
import { OutletWithContext, useOutletContext } from '@/lib/router-helpers';
import useCloud from '@/hooks/use-cloud';
import { useEffect, useMemo } from 'react';

function Main() {
  const ctx = useOutletContext<UserContextType & MembershipsContextType>();
  const { user, memberships } = ctx;
  const { tenantId } = useCurrentTenantId();
  const { cloud, featureFlags } = useCloud(tenantId);
  const managedWorkerEnabled = featureFlags?.['managed-worker'] === 'true';

  const navSections = useMemo(
    () =>
      sideNavItems({
        canBill: cloud?.canBill,
        managedWorkerEnabled,
      }),
    [cloud?.canBill, managedWorkerEnabled],
  );

  const childCtx = useContextFromParent({
    user,
    memberships,
  });

  useEffect(() => {
    console.log('[MainV1] render', {
      path: window.location.pathname,
      tenantId,
      hasUser: Boolean(user),
      membershipsCount: memberships?.length,
      canBill: cloud?.canBill,
      managedWorkerEnabled,
      navSectionCount: navSections.length,
    });
  }, [
    tenantId,
    user,
    memberships?.length,
    cloud?.canBill,
    managedWorkerEnabled,
    navSections.length,
  ]);

  return (
    <ThreeColumnLayout
      sidebar={<SideNav navItems={navSections} />}
      sidePanel={<SidePanel />}
      // mainClassName="overflow-auto"
      mainContainerType="inline-size"
    >
      {/* TODO-DESIGN: replace the color with a tailwind color */}
      {/* NOTE: shadow is used instead of border to avoid dom layout shift within inline-size containers */}
      <div className="shadow-[inset_1px_0_0_0_hsl(var(--border)/0.5),inset_0_1px_0_0_hsl(var(--border)/0.5)] md:rounded-tl-xl p-4 h-full dark:bg-[#050A23] bg-background overflow-y-auto [scrollbar-gutter:stable_both-edges]">
        <OutletWithContext context={childCtx} />
      </div>
    </ThreeColumnLayout>
  );
}

export default Main;
