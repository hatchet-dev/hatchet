import React from 'react';
import { Spinner } from '@/components/v1/ui/loading';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

export const VersionInfo: React.FC = () => {
  const { data, isLoading, isError, error } = useQuery({
    ...queries.info.getVersion,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center">
        <Spinner />
      </div>
    );
  }

  if (isError || !data?.version) {
    const errorMessage =
      error instanceof Error ? error.message : 'Failed to fetch version info';
    return <div className="text-red-500 text-xs">{errorMessage}</div>;
  }

  return <div className="text-xs">{data.version}</div>;
};
