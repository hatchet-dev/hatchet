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
import useCloud from '@/pages/auth/hooks/use-cloud';
import { useMemo } from 'react';

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
