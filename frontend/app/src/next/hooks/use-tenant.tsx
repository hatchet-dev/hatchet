import {
  useCallback,
  useMemo,
  useState,
  createContext,
  useContext,
  ReactNode,
  useEffect,
} from 'react';
import api, {
  UpdateTenantRequest,
  Tenant,
  TenantMember,
  CreateTenantRequest,
  TenantResourceLimit,
} from '@/lib/api';
import useUser from './use-user';
import { useSearchParams } from 'react-router-dom';
import {
  useMutation,
  UseMutationResult,
  useQuery,
  useQueryClient,
  UseQueryResult,
} from '@tanstack/react-query';
import { useToast } from './utils/use-toast';

interface TenantState {
  tenant?: Tenant;
  membership?: TenantMember['role'];
  limit?: UseQueryResult<TenantResourceLimit[], Error>;
  isLoading: boolean;
  setTenant: (tenantId: string) => void;
  create: UseMutationResult<Tenant, Error, string, unknown>;
  update: {
    mutate: (data: UpdateTenantRequest) => void;
    mutateAsync: (data: UpdateTenantRequest) => Promise<Tenant>;
    isPending: boolean;
  };
}

const TenantContext = createContext<TenantState | null>(null);

export function clearTenant() {
  localStorage.removeItem('tenant');
}

interface TenantProviderProps {
  children: ReactNode;
}

export function TenantProvider({ children }: TenantProviderProps) {
  const { memberships, isLoading: isUserLoading } = useUser();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [currentTenantId, setCurrentTenantId] = useState<string | undefined>(
    () =>
      searchParams.get('tenant') ?? localStorage.getItem('tenant') ?? undefined,
  );

  // make sure the tenant id is in the url if it's not already
  useEffect(() => {
    if (!searchParams.get('tenant') && currentTenantId) {
      const newSearchParams = new URLSearchParams(searchParams);
      newSearchParams.set('tenant', currentTenantId);
      setSearchParams(newSearchParams, { replace: true });
    }
  }, [searchParams, currentTenantId, setSearchParams]);

  const setTenant = useCallback(
    (tenantId?: string) => {
      if (!tenantId) {
        return;
      }

      const newSearchParams = new URLSearchParams(searchParams);
      newSearchParams.set('tenant', tenantId);
      setSearchParams(newSearchParams, { replace: true });
      setCurrentTenantId(tenantId);
      localStorage.setItem('tenant', tenantId);

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
    if (!currentTenantId) {
      return undefined;
    }

    return memberships?.find(
      (membership) => membership.tenant?.metadata.id === currentTenantId,
    );
  }, [currentTenantId, memberships]);

  // Handle setting initial tenant when no tenant is selected
  useEffect(() => {
    if (!currentTenantId && memberships?.[0]?.tenant?.metadata.id) {
      setTenant(memberships[0].tenant.metadata.id);
    }
  }, [currentTenantId, memberships, setTenant]);

  const tenant = membership?.tenant;

  // Mutation for creating a tenant
  const createTenantMutation = useMutation({
    mutationKey: ['tenant:create'],
    mutationFn: async (name: string): Promise<Tenant> => {
      try {
        const tenantData: CreateTenantRequest = {
          name,
          slug: name.toLowerCase().replace(/\s+/g, '-'),
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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user:*'] });
      queryClient.invalidateQueries({ queryKey: ['tenant:*'] });
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

  const value = {
    tenant,
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

  return (
    <TenantContext.Provider value={value}>{children}</TenantContext.Provider>
  );
}

export default function useTenant(): TenantState {
  const context = useContext(TenantContext);
  if (context === null) {
    throw new Error('useTenant must be used within a TenantProvider');
  }
  return context;
}
