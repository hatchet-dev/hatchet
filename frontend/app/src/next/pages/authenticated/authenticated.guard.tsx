import AnalyticsProvider from '@/next/components/providers/analytics.provider';
import SupportChat from '@/next/components/providers/support-chat.provider';
import useUser from '@/next/hooks/use-user';
import { ROUTES } from '@/next/lib/routes';
import { Navigate, Outlet, useLocation } from 'react-router-dom';

export default function AuthenticatedGuard() {
  const user = useUser();
  const location = useLocation();

  // user is not authenticated
  if (!user.isLoading && !user.data) {
    return <Navigate to="../auth/login" />;
  }

  // user email is not verified
  if (!user.isLoading && !user.data?.emailVerified) {
    return <Navigate to="../auth/verify-email" />;
  }

  // user has no tenant
  if (
    !user.isLoading &&
    user.memberships &&
    user.memberships.length === 0 &&
    !location.pathname.startsWith('/onboarding')
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
