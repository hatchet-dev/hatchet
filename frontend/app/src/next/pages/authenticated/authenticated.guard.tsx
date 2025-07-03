import { TenantUIVersion, TenantVersion } from '@/lib/api';
import AnalyticsProvider from '@/next/components/providers/analytics.provider';
import SupportChat from '@/next/components/providers/support-chat.provider';
import { useTenantDetails } from '@/next/hooks/use-tenant';
import useUser from '@/next/hooks/use-user';
import { ROUTES } from '@/next/lib/routes';
import { useEffect } from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';

export default function AuthenticatedGuard() {
  const user = useUser();
  const location = useLocation();
  const { tenant, isLoading: tenantIsLoading } = useTenantDetails();

  useEffect(() => {
    if (
      !user.isLoading &&
      !tenantIsLoading &&
      tenant &&
      tenant?.uiVersion !== TenantUIVersion.V1 &&
      tenant.uiVersion
    ) {
      if (tenant.version === TenantVersion.V0) {
        window.location.href = `/workflow-runs?tenant=${tenant?.metadata.id}`;
      } else {
        window.location.href = `/v1/runs?tenant=${tenant?.metadata.id}`;
      }
    }
  });
  // user is not authenticated
  if (!user.isLoading && !user.data) {
    return <Navigate to="../auth/login" />;
  }

  // user email is not verified
  if (!user.isLoading && !user.data?.emailVerified) {
    return <Navigate to="../auth/verify-email" />;
  }

  if (
    !user.isLoading &&
    user.memberships &&
    user.memberships.length === 0 &&
    !location.pathname.startsWith('/next/onboarding')
  ) {
    return <Navigate to={ROUTES.onboarding.newTenant} />;
  }

  if (!user) {
    return null;
  }

  return (
    <AnalyticsProvider>
      <SupportChat>
        <Outlet />
      </SupportChat>
    </AnalyticsProvider>
  );
}
