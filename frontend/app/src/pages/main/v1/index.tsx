import { SideNav } from '../../../components/v1/nav/side-nav';
import { sideNavItems } from './side-nav-items';
import { useTheme } from '@/components/hooks/use-theme';
import { ThreeColumnLayout } from '@/components/layout/three-column-layout';
import { SidePanel } from '@/components/v1/nav/side-panel';
import useCloud from '@/hooks/use-cloud';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import {
  MembershipsContextType,
  UserContextType,
  useContextFromParent,
} from '@/lib/outlet';
import { OutletWithContext, useOutletContext } from '@/lib/router-helpers';
import { useUserUniverse } from '@/providers/user-universe';
import { usePrefetchQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

function Main() {
  const ctx = useOutletContext<UserContextType & MembershipsContextType>();
  const { user, memberships } = ctx;
  const { tenantId } = useCurrentTenantId();
  const { cloud, featureFlags, isCloudEnabled } = useCloud(tenantId);
  const managedWorkerEnabled = featureFlags?.['managed-worker'] === 'true';

  usePrefetchQuery(queries.workflows.list(tenantId, { limit: 200 }));

  const { toggleTheme, currentlyVisibleTheme } = useTheme();
  const { logoutMutation } = useUserUniverse();

  const navSections = useMemo(
    () =>
      sideNavItems({
        canBill: cloud?.canBill,
        managedWorkerEnabled,
        isCloudEnabled,
        onToggleTheme: toggleTheme,
        currentlyVisibleTheme,
      }),
    [
      cloud?.canBill,
      managedWorkerEnabled,
      isCloudEnabled,
      toggleTheme,
      currentlyVisibleTheme,
    ],
  );

  const childCtx = useContextFromParent({
    user,
    memberships,
  });

  return (
    <ThreeColumnLayout
      sidebar={
        <SideNav navItems={navSections} onLogout={logoutMutation.mutate} />
      }
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
