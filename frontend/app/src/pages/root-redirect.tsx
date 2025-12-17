import { useNavigate } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useEffect } from 'react';
import { appRoutes } from '@/router';
import { useAtom } from 'jotai';
import { lastTenantAtom } from '@/lib/atoms';

export default function RootRedirect() {
  const navigate = useNavigate();
  const [lastTenant] = useAtom(lastTenantAtom);

  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
    retry: false,
  });

  useEffect(() => {
    if (lastTenant) {
      navigate({
        to: appRoutes.tenantRunsRoute.to,
        params: { tenant: lastTenant.metadata.id },
        replace: true,
      });

      return;
    }

    if (
      listMembershipsQuery.data?.rows &&
      listMembershipsQuery.data.rows.length > 0
    ) {
      const firstTenant = listMembershipsQuery.data.rows[0].tenant;
      if (firstTenant) {
        navigate({
          to: appRoutes.tenantRoute.to,
          params: { tenant: firstTenant.metadata.id },
          replace: true,
        });
      }
    }
  }, [listMembershipsQuery.data, navigate, lastTenant]);

  return <Loading />;
}
