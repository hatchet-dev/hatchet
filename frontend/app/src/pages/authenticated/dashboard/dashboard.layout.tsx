import useUser from '@/hooks/use-user';
import { Navigate, Outlet } from 'react-router-dom';

export default function Authenticated() {
  const user = useUser();

  // user is not authenticated
  if (!user.isLoading && !user.data) {
    return <Navigate to="/auth/login" />;
  }

  return (
    <>
      {user && (
        <div className="flex flex-col h-screen bg-yellow-500">
          <Outlet />
        </div>
      )}
    </>
  );
}
