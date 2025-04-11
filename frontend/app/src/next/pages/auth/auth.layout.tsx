import { PropsWithChildren } from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import useUser from '@/next/hooks/use-user';
import { CenterStageLayout } from '@/next/components/layouts/center-stage.layout';
import { ROUTES } from '@/next/lib/routes';

export default function AuthLayoutGuard({ children }: PropsWithChildren) {
  const user = useUser();

  if (user.data && user.data.emailVerified) {
    return <Navigate to={ROUTES.runs.list} />;
  }

  return <CenterStageLayout>{children ?? <Outlet />}</CenterStageLayout>;
}
