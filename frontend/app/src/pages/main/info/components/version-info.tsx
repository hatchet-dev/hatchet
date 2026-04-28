import { Spinner } from '@/components/v1/ui/loading';
import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import React from 'react';

export const VersionInfo: React.FC = () => {
  const { isControlPlaneEnabled } = useControlPlane();
  const { data, isLoading, isError, error } = useQuery({
    ...queries.info.getVersion,
    enabled: !isControlPlaneEnabled,
  });

  const { isCloudEnabled } = useCloud();

  if (isControlPlaneEnabled) {
    return null;
  }

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
