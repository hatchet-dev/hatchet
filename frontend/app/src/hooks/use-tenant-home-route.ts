import api from '@/lib/api';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';

/**
 * Hook to determine the correct home route for a tenant
 * Returns overview route if tenant has no workflows, runs route if workflows exist
 */
export function useTenantHomeRoute(tenantId: string | undefined) {
  const workflowsQuery = useQuery({
    queryKey: ['tenant-home-route', tenantId],
    queryFn: async () => {
      if (!tenantId) {
        return { hasWorkflows: false };
      }

      try {
        const response = await api.workflowList(tenantId, { limit: 1 });
        const hasWorkflows =
          response.data.rows && response.data.rows.length > 0;
        return { hasWorkflows };
      } catch (error) {
        // On error, default to runs page
        return { hasWorkflows: true };
      }
    },
    enabled: !!tenantId,
    staleTime: 60000, // Cache for 1 minute
    retry: false,
  });

  // Default to runs route while loading or if no data
  const homeRoute =
    (workflowsQuery.data?.hasWorkflows ?? true)
      ? appRoutes.tenantRunsRoute.to
      : appRoutes.tenantOverviewRoute.to;

  return {
    homeRoute,
    isLoading: workflowsQuery.isLoading,
  };
}
