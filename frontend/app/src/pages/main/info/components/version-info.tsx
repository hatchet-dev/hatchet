import React, { useEffect, useState } from 'react';
// import api from '@/lib/api';
import { Spinner } from '@/components/ui/loading';
import api from '@/lib/api';

export const VersionInfo: React.FC = () => {
  const [version, setVersion] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const fetchVersion = async () => {
    try {
      const response = await api.infoGetVersion();
      setVersion(response.data.version || 'Unknown');
    } catch (err) {
      setError('Failed to fetch version info');
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchVersion();
  }, []);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center">
        <Spinner />
      </div>
    );
  }

  if (error) {
    return <div className="text-red-500 text-xs">{error}</div>;
  }

  return (
    <div className="text-sm">
      <p>
        <span className="text-xs"> {version}</span>
      </p>
    </div>
  );
};
