import api from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useMemo, useState } from 'react';
import { useToast } from './utils/use-toast';
import { useTenantDetails } from './use-tenant';

export default function useApiMeta() {
  const [refetchInterval, setRefetchInterval] = useState<number | undefined>(
    undefined,
  );
  const { toast } = useToast();

  const { tenant } = useTenantDetails();

  const metaQuery = useQuery({
    queryKey: ['metadata:get'],
    queryFn: async () => {
      try {
        return (await api.metadataGet()).data;
      } catch (error) {
        toast({
          title: 'Error fetching metadata',

          variant: 'destructive',
        });
        throw error;
      }
    },
    staleTime: 1000 * 60 * 60, // 1 hour
    retryDelay: 150,
    retry: 2,
    refetchInterval,
  });

  const integrationsQuery = useQuery({
    queryKey: ['metadata:get:integrations'],
    queryFn: async () => {
      try {
        const meta = await api.metadataListIntegrations();
        return meta.data;
      } catch (error) {
        toast({
          title: 'Error fetching integrations',

          variant: 'destructive',
        });
        throw error;
      }
    },
    enabled: !!tenant,
  });

  const { data: version } = useQuery({
    queryKey: ['info:version'],
    queryFn: async () => {
      try {
        return (await api.infoGetVersion()).data;
      } catch (error) {
        toast({
          title: 'Error fetching version info',

          variant: 'destructive',
        });
        throw error;
      }
    },
    retryDelay: 150,
    retry: 2,
    refetchInterval,
  });

  const { data: cloudMeta } = useQuery({
    queryKey: ['cloud-metadata:get'],
    queryFn: async () => {
      try {
        const meta = (await cloudApi.metadataGet()).data;
        return meta;
      } catch (e) {
        toast({
          title: 'Error fetching cloud metadata',

          variant: 'destructive',
        });
        return;
      }
    },
    staleTime: 1000 * 60,
    retryDelay: 150,
    retry: 2,
    refetchInterval,
  });

  useEffect(() => {
    setRefetchInterval(metaQuery.isError ? 15000 : undefined);
  }, [metaQuery.isError]);

  const isCloud = useMemo(() => !(cloudMeta as any)?.errors, [cloudMeta]);

  return {
    oss: metaQuery.data,
    integrations: integrationsQuery.data,
    cloud: cloudMeta,
    isLoading: metaQuery.isLoading,
    hasFailed: metaQuery.isError && metaQuery.error,
    refetchInterval,
    version: version?.version,
    isCloud,
  };
}
