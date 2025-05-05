import api from '@/lib/api';
import useTenant from '@/next/hooks/use-tenant';
import { useQuery } from '@tanstack/react-query';

export default function EventsPage() {
  const { tenant } = useTenant();

  const { data, isLoading } = useQuery({
    queryKey: ['v1:events:list', tenant],
    queryFn: async () => {
      try {
        return (
          await api.v1EventList(tenant?.metadata.id || '', {
            offset: 0,
            limit: 10,
          })
        ).data;
      } catch (error) {
        return { rows: [] };
      }
    },
  });

  if (!tenant) {
    return (
      <div className="flex-grow h-full w-full flex items-center justify-center">
        <p>Loading tenant information...</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex-grow h-full w-full flex items-center justify-center">
        <p>Loading events</p>
      </div>
    );
  }

  const events = data?.rows || [];

  return (
    <div>
      {events.map((e) => (
        <div>
          <h3>{e.key}</h3>
          <h3>{JSON.stringify(e.additionalMetadata || '{}')}</h3>
          <h3>{e.metadata.id}</h3>
          <p>{e.metadata.createdAt}</p>
          <p>{e.metadata.updatedAt}</p>
        </div>
      ))}
    </div>
  );
}
