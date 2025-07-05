import { useCallback, useMemo } from 'react';
import api, {
  UpdateTenantRequest,
  Tenant,
  CreateTenantRequest,
  TenantUIVersion,
  TenantVersion,
} from '@/lib/api';
import useUser from './use-user';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useToast } from './utils/use-toast';
import invariant from 'tiny-invariant';

export function useCurrentTenantId() {
  const params = useParams();
  const tenantId = params.tenantId;

  invariant(tenantId, 'Tenant ID is required');

  return { tenantId };
}

export function useTenantDetails() {
  const params = useParams();
  const tenantId = params.tenantId;

  const { memberships, isLoading: isUserLoading } = useUser();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const location = useLocation();
  const navigate = useNavigate();

  const setTenant = useCallback(
    (tenantId?: string) => {
      const currentPath = location.pathname;
      const targetTenant = memberships?.find(
        (m) => m.tenant?.metadata.id === tenantId,
      )?.tenant;

      const targetEngineVersion = targetTenant?.version || TenantVersion.V1;
      const targetUIVersion = targetTenant?.uiVersion || TenantUIVersion.V1;

      if (targetUIVersion === TenantUIVersion.V1) {
        const newPath = currentPath.replace(
          /\/tenants\/([^/]+)/,
          `/tenants/${tenantId}`,
        );

        navigate(newPath);
      } else if (targetEngineVersion === TenantVersion.V1) {
        window.location.href = `/tenants/${tenantId}/runs`;
      } else if (targetEngineVersion === TenantVersion.V0) {
        window.location.href = `/workflow-runs?tenant=${tenantId}`;
      }
    },
    [navigate, location.pathname, memberships],
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

  // Mutation for creating a tenant
  const createTenantMutation = useMutation({
    mutationKey: ['tenant:create'],
    mutationFn: async ({
      name,
      uiVersion,
    }: {
      name: string;
      uiVersion: TenantUIVersion;
    }): Promise<Tenant> => {
      try {
        const tenantData: CreateTenantRequest = {
          name,
          slug: name.toLowerCase().replace(/\s+/g, '-'),
          uiVersion,
        };

        const response = await api.tenantCreate(tenantData);
        return response.data;
      } catch (error) {
        toast({
          title: 'Error creating tenant',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: async (data) => {
      await queryClient.invalidateQueries({ queryKey: ['user:*'] });
      if (data.metadata.id) {
        setTenant(data.metadata.id);
      }
      return data;
    },
  });

  // Mutation for updating tenant details
  const updateTenantMutation = useMutation({
    mutationKey: ['tenant:update', tenant?.metadata.id],
    mutationFn: async (data: UpdateTenantRequest) => {
      if (!tenant?.metadata.id) {
        throw new Error('Tenant not found');
      }
      try {
        const response = await api.tenantUpdate(tenant.metadata.id, data);
        return response.data;
      } catch (error) {
        toast({
          title: 'Error updating tenant',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['user:*'] });
      await queryClient.invalidateQueries({ queryKey: ['tenant:*'] });
    },
  });

  const resourcePolicyQuery = useQuery({
    queryKey: ['tenant-resource-policy:get', tenant?.metadata.id],
    queryFn: async () => {
      try {
        return (await api.tenantResourcePolicyGet(tenant?.metadata.id ?? ''))
          .data.limits;
      } catch (error) {
        toast({
          title: 'Error fetching tenant resource policy',

          variant: 'destructive',
          error,
        });
        return [];
      }
    },
    enabled: !!tenant?.metadata.id,
  });

  return {
    tenantId,
    tenant,
    defaultTenant,
    isLoading: isUserLoading,
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
