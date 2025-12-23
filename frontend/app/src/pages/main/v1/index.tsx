import { SideNav } from '../../../components/v1/nav/side-nav';
import { sideNavItems } from './side-nav-items';
import { ThreeColumnLayout } from '@/components/layout/three-column-layout';
import { SidePanel } from '@/components/v1/nav/side-panel';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { SidePanelProvider } from '@/hooks/use-side-panel';
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

  if (!user || !memberships) {
    return <Loading />;
  }

  return (
    <SidePanelProvider>
      <ThreeColumnLayout
        sidebar={<SideNav navItems={navSections} />}
        sidePanel={<SidePanel />}
        mainClassName="overflow-auto"
        mainContainerType="inline-size"
      >
        {/* TODO-DESIGN: replace the color with a tailwind color */}
        <div className="border-l border-t md:rounded-tl-xl p-4 h-full dark:bg-[#050A23] bg-background ">
          <OutletWithContext context={childCtx} />
        </div>
      </ThreeColumnLayout>
    </SidePanelProvider>
  );
}

export default Main;
