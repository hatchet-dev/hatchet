import AnalyticsProvider from '@/components/providers/analytics.provider';
import SupportChat from '@/components/providers/support-chat.provider';
import useUser from '@/hooks/use-user';
import { Navigate, Outlet } from 'react-router-dom';

export default function Authenticated() {
  const user = useUser();

  if (!user.isLoading && !user.data) {
    return <Navigate to="/auth/login" />;
  }

  // if (!userQuery.data) {
  //   return <Navigate to="/auth/login" />;
  // }

  console.log(user);

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
