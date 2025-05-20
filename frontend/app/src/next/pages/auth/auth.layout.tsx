import { PropsWithChildren } from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import useUser from '@/next/hooks/use-user';
import { CenterStageLayout } from '@/next/components/layouts/center-stage.layout';
import { ROUTES } from '@/next/lib/routes';
import { useTenant } from '@/next/hooks/use-tenant';

export default function AuthLayoutGuard({ children }: PropsWithChildren) {
  const user = useUser();
  const { tenant } = useTenant();

  if (user.data && tenant?.metadata.id && user.data.emailVerified) {
    return <Navigate to={ROUTES.runs.list(tenant?.metadata.id)} />;
  }

  return <CenterStageLayout>{children ?? <Outlet />}</CenterStageLayout>;
}
