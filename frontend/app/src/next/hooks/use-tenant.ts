import {
  Tenant,
  TenantMember,
  CreateTenantRequest,
  UpdateTenantRequest,
} from '@/lib/api';
import api from '@/lib/api/api';
import useUser from './use-user';
import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  useMutation,
  UseMutationResult,
  useQueryClient,
} from '@tanstack/react-query';

interface TenantState {
  tenant?: Tenant;
  membership?: TenantMember['role'];
  isLoading: boolean;
  setTenant: (tenantId: string) => void;
  create: UseMutationResult<Tenant, Error, string, unknown>;
  update: {
    mutate: (data: UpdateTenantRequest) => void;
    isPending: boolean;
  };
}

export default function useTenant(): TenantState {
  const { memberships, isLoading: isUserLoading } = useUser();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();

  const setTenant = useCallback(
    (tenantId: string) => {
      const newSearchParams = new URLSearchParams(searchParams);
      newSearchParams.set('tenant', tenantId);
      setSearchParams(newSearchParams, { replace: true });

      // Get the previous tenant ID that might be in existing query keys
      const prevTenantId = searchParams.get('tenant');

      // Invalidate all queries that use the tenant ID in their query key
      if (prevTenantId) {
        queryClient.invalidateQueries({
          predicate: (query) => {
            if (Array.isArray(query.queryKey)) {
              return query.queryKey.includes(prevTenantId);
            }
            return false;
          },
        });
      }
    },
    [searchParams, setSearchParams, queryClient],
  );

  const membership = useMemo(() => {
    const tenantId = searchParams.get('tenant');

    if (tenantId == null) {
      const fallback = memberships?.[0];
      if (!fallback?.tenant?.metadata.id) {
        return;
      }
      setTenant(fallback.tenant.metadata.id);
      return fallback;
    }

    const matched = memberships?.find(
      (membership) => membership.tenant?.metadata.id === tenantId,
    );

    if (!matched?.tenant?.metadata.id) {
      return;
    }

    return matched;
  }, [memberships, searchParams, setTenant]);

  const tenant = membership?.tenant;

  // Mutation for creating a tenant
  const createTenantMutation = useMutation({
    mutationKey: ['tenant:create'],
    mutationFn: async (name: string): Promise<Tenant> => {
      const tenantData: CreateTenantRequest = {
        name,
        slug: name,
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

  // Mutation for updating tenant details
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
      window.location.reload();
    },
  });

  return {
    tenant,
    isLoading: isUserLoading,
    membership: membership?.role,
    setTenant,
    create: createTenantMutation,
    update: {
      mutate: (data: UpdateTenantRequest) => updateTenantMutation.mutate(data),
      isPending: updateTenantMutation.isPending,
    },
  };
}
