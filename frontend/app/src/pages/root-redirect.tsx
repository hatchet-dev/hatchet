import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useEffect } from 'react';

export default function RootRedirect() {
  const navigate = useNavigate();

  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
    retry: false,
  });

  useEffect(() => {
    if (
      listMembershipsQuery.data?.rows &&
      listMembershipsQuery.data.rows.length > 0
    ) {
      const firstTenant = listMembershipsQuery.data.rows[0].tenant;
      if (firstTenant) {
        navigate(`/tenants/${firstTenant.metadata.id}`, {
          replace: true,
        });
      }
    }
  }, [listMembershipsQuery.data, navigate]);

  return <Loading />;
}
