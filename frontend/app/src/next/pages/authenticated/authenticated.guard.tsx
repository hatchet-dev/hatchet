import { TenantUIVersion } from '@/lib/api';
import AnalyticsProvider from '@/next/components/providers/analytics.provider';
import SupportChat from '@/next/components/providers/support-chat.provider';
import { useTenantDetails } from '@/next/hooks/use-tenant';
import useUser from '@/next/hooks/use-user';
import { ROUTES } from '@/next/lib/routes';
import { Navigate, Outlet, useLocation } from 'react-router-dom';

export default function AuthenticatedGuard() {
  const user = useUser();
  const location = useLocation();
  const { tenant, isLoading: tenantIsLoading } = useTenantDetails();

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
    !tenantIsLoading &&
    tenant &&
    tenant?.uiVersion !== TenantUIVersion.V1
  ) {
    return <Navigate to={'/v1/runs'} />;
  }

  if (
    !user.isLoading &&
    user.memberships &&
    user.memberships.length === 0 &&
    !location.pathname.startsWith('/next/onboarding')
  ) {
    return <Navigate to={ROUTES.onboarding.newTenant} />;
  }
  return (
    <>
      {user && (
        <>
          <AnalyticsProvider>
            <SupportChat>
              <Outlet />
            </SupportChat>
          </AnalyticsProvider>
        </>
      )}
    </>
  );
}
