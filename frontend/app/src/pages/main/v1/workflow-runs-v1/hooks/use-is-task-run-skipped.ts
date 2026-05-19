import { queries, V1TaskEventType } from '@/lib/api';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';

type UseIsTaskRunSkippedProps = {
  taskRunId: string;
  limit?: number;
};

export const useIsTaskRunSkipped = ({
  taskRunId,
  limit = 50,
}: UseIsTaskRunSkippedProps) => {
  const { tenant } = useParams({ from: appRoutes.tenantRoute.to });
  const eventsQuery = useQuery({
    ...queries.v1TaskEvents.list(tenant, { limit, offset: 0 }, taskRunId),
  });

  const isSkipped =
    eventsQuery.data?.rows?.some(
      (event) => event.eventType === V1TaskEventType.SKIPPED,
    ) ?? false;

  return {
    isSkipped,
    isLoading: eventsQuery.isLoading,
  };
};
