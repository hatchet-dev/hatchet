import {
  Alert,
  AlertTitle,
  AlertDescription,
} from '@/next/components/ui/alert';
import { Button } from '@/next/components/ui/button';
import useUser from '@/next/hooks/use-user';
import { useState } from 'react';
import { useEffect } from 'react';
import { FaEnvelope } from 'react-icons/fa';
import { Navigate } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import useTenant from '@/next/hooks/use-tenant';
export default function VerifyEmailPage() {
  const retryInterval = 5000;

  const user = useUser({
    refetchInterval: retryInterval,
  });
  const { tenant } = useTenant();

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

  // once the email is verified, redirect to the home page
  if (user.data?.emailVerified) {
    return <Navigate to={ROUTES.runs.list(tenant?.metadata.id || '')} />;
  }

  return (
    <div className="max-w-md mx-auto">
      <Alert>
        <FaEnvelope className="h-4 w-4" />
        <AlertTitle className="text-lg">Email verification required</AlertTitle>
        <AlertDescription>
          Please contact your Hatchet administrator to verify your account for{' '}
          {user.data?.email}
          <br />
          <div className="text-sm text-gray-500">
            {!user.isLoading && countdown
              ? `Retrying in ${countdown} seconds...`
              : 'Retrying...'}
          </div>
        </AlertDescription>
      </Alert>

      <div className="flex justify-end pt-4">
        <Button
          variant="ghost"
          onClick={() => user.logout.mutate()}
          loading={user.logout.isPending}
        >
          Log Out
        </Button>
      </div>
    </div>
  );
}
