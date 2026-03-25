import { Spinner } from '@/components/v1/ui/loading';
import { queries } from '@/lib/api';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { useQuery } from '@tanstack/react-query';
import React from 'react';

export const VersionInfo: React.FC = () => {
  const { data, isLoading, isError, error } = useQuery({
    ...queries.info.getVersion,
  });

  const { isCloudEnabled } = useCloud();

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
    return <div className="text-xs text-red-500">{errorMessage}</div>;
  }

  return (
    <div className="text-xs">
      {isCloudEnabled ? 'Cloud' : 'Self-Hosted'} {data.version}
    </div>
  );
};
