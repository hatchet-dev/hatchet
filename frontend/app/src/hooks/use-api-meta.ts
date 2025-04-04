import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

export default function useApiMeta() {
  const metaQuery = useQuery({
    ...queries.metadata.get,
  });

  if (metaQuery.isError) {
    // TODO: handle error
    console.error(metaQuery.error);
  }

  return {
    data: metaQuery.data,
    isLoading: false,
  };
}
