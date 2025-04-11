import { CenterStageLayout } from '@/next/components/layouts/center-stage.layout';
import { useEffect, useState } from 'react';
import { IoCloudOfflineSharp } from 'react-icons/io5';
import { Alert, AlertTitle, AlertDescription } from '../ui/alert';

interface ApiConnectionErrorProps {
  retryInterval?: number;
  loading?: boolean;
  errorMessage?: string;
}

export function ApiConnectionError({
  retryInterval,
  errorMessage,
}: ApiConnectionErrorProps) {
  const [countdown, setCountdown] = useState<number | null>(null);

  // TODO countdown thing
  useEffect(() => {
    if (!retryInterval) {
      setCountdown(null);
      return;
    }

    setCountdown(Math.ceil(retryInterval / 1000));
    const interval = setInterval(() => {
      setCountdown((prev) => {
        if (prev === null || prev <= 1) {
          clearInterval(interval);
          return null;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [retryInterval]);

  return (
    <CenterStageLayout>
      <div className="max-w-md mx-auto">
        <Alert variant="destructive">
          <IoCloudOfflineSharp className="h-4 w-4" />
          <AlertTitle className="text-lg">Can't connect to API</AlertTitle>
          <AlertDescription>
            Hatchet cannot connect to the API. Please check your connection and
            try again.
            <div className="py-10">
              <code>{errorMessage}</code>
            </div>
            <br />
            <div className="text-sm text-gray-500">
              {countdown
                ? `Retrying in ${countdown} seconds...`
                : 'Retrying...'}
            </div>
          </AlertDescription>
        </Alert>
      </div>
    </CenterStageLayout>
  );
}
