import { PropsWithChildren } from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import useUser from '@/next/hooks/use-user';
import { CenterStageLayout } from '@/next/components/layouts/center-stage.layout';

export default function AuthLayoutGuard({ children }: PropsWithChildren) {
  const user = useUser();

  if (user.data && user.data.emailVerified) {
    return <Navigate to="/" />;
  }

  return <CenterStageLayout>{children ?? <Outlet />}</CenterStageLayout>;
}
