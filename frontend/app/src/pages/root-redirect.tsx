import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useEffect } from 'react';
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
      navigate(`/tenants/${lastTenant.metadata.id}`, {
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
        navigate(`/tenants/${firstTenant.metadata.id}`, {
          replace: true,
        });

        return;
      }

      return;
    }
  }, [listMembershipsQuery.data, navigate]);

  return <Loading />;
}
