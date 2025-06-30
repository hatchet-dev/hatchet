import { useCallback, useMemo } from 'react';
import api, {
  UpdateTenantRequest,
  Tenant,
  CreateTenantRequest,
  TenantUIVersion,
  queries,
} from '@/lib/api';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import invariant from 'tiny-invariant';
import { useToast } from './use-toast';

export function useCurrentTenantId() {
  const params = useParams();
  const tenantId = params.tenantId;

  invariant(tenantId, 'Tenant ID is required');

  return { tenantId };
}

export function useTenantDetails() {
  const params = useParams();
  const tenantId = params.tenantId;

  const membershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  const memberships = useMemo(
    () => membershipsQuery.data?.rows || [],
    [membershipsQuery.data],
  );

  const queryClient = useQueryClient();
  const { toast } = useToast();
  const location = useLocation();
  const navigate = useNavigate();

  const setTenant = useCallback(
    (tenantId?: string) => {
      const currentPath = location.pathname;

      const newPath = currentPath.replace(
        /\/tenants\/([^/]+)/,
        `/tenants/${tenantId}`,
      );

      navigate(newPath);
    },
    [navigate, location.pathname],
  );

  const membership = useMemo(() => {
    if (!tenantId) {
      return undefined;
    }

    return memberships?.find(
      (membership) => membership.tenant?.metadata.id === tenantId,
    );
  }, [tenantId, memberships]);

  const tenant = membership?.tenant;
  const defaultTenant = memberships?.[0]?.tenant;

  const createTenantMutation = useMutation({
    mutationKey: ['tenant:create'],
    mutationFn: async ({
      name,
      uiVersion,
    }: {
      name: string;
      uiVersion: TenantUIVersion;
    }): Promise<Tenant> => {
      const tenantData: CreateTenantRequest = {
        name,
        slug: name.toLowerCase().replace(/\s+/g, '-'),
        uiVersion,
      };

      const response = await api.tenantCreate(tenantData);
      return response.data;
    },
    onSuccess: async (data) => {
      await queryClient.invalidateQueries({ queryKey: ['user:*'] });
      if (data.metadata.id) {
        setTenant(data.metadata.id);
      }
      return data;
    },
  });

  const updateTenantMutation = useMutation({
    mutationKey: ['tenant:update', tenant?.metadata.id],
    mutationFn: async (data: UpdateTenantRequest) => {
      if (!tenant?.metadata.id) {
        throw new Error('Tenant not found');
      }
      const response = await api.tenantUpdate(tenant.metadata.id, data);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user:*'] });
      queryClient.invalidateQueries({ queryKey: ['tenant:*'] });
    },
  });

  const resourcePolicyQuery = useQuery({
    queryKey: ['tenant-resource-policy:get', tenant?.metadata.id],
    queryFn: async () => {
      return (await api.tenantResourcePolicyGet(tenant?.metadata.id ?? '')).data
        .limits;
    },
    enabled: !!tenant?.metadata.id,
  });

  return {
    tenantId,
    tenant,
    defaultTenant,
    isLoading: membershipsQuery.isLoading,
    membership: membership?.role,
    setTenant,
    create: createTenantMutation,
    update: {
      mutate: (data: UpdateTenantRequest) => updateTenantMutation.mutate(data),
      mutateAsync: (data: UpdateTenantRequest) =>
        updateTenantMutation.mutateAsync(data),
      isPending: updateTenantMutation.isPending,
    },
    limit: resourcePolicyQuery,
  };
}
