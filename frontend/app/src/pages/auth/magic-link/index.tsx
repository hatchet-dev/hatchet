import { AuthLayout } from '../components/auth-layout';
import { AuthLegalText } from '../components/auth-legal-text';
import { accountSwitchMessage, isTerminalMagicLinkError } from './magic-link';
import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { mfaStatus } from '@/lib/api/mfa';
import { useUserApi } from '@/lib/api/user-wrapper';
import { useApiError } from '@/lib/hooks';
import { initialMagicLinkToken } from '@/lib/magic-link-bootstrap';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import { isAxiosError } from 'axios';
import { useState } from 'react';

export default function MagicLinkConfirm() {
  const token = initialMagicLinkToken;
  const [errors, setErrors] = useState<string[]>(
    token ? [] : ['This login link is missing or invalid.'],
  );
  const [retryErrors, setRetryErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors });
  const { handleApiError: handleRetryableError } = useApiError({
    setErrors: setRetryErrors,
  });
  const { invalidate: invalidateUserUniverse } = useUserUniverse();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { userGetCurrentQuery, userUpdateMagicLinkMutation } = useUserApi();
  const currentUser = useQuery({
    ...userGetCurrentQuery(),
    retry: false,
  });
  const confirm = useMutation({
    ...userUpdateMagicLinkMutation(),
    onMutate: () => setRetryErrors([]),
    onSuccess: async (user) => {
      if ((await mfaStatus()).pending) {
        await navigate({ to: appRoutes.authMFARoute.to });
        return;
      }
      // The signed-out lookup on this page leaves an error cached under the
      // current-user key, and the app shell reads that key without refetching.
      queryClient.setQueryData(userGetCurrentQuery().queryKey, user);
      await invalidateUserUniverse();
      await new Promise((resolve) => setTimeout(resolve, 0));
      navigate({ to: appRoutes.authenticatedRoute.to });
    },
    onError: (error) => {
      if (!isAxiosError(error)) {
        setRetryErrors(['Something went wrong.']);
        return;
      }
      if (isTerminalMagicLinkError(error.response?.status)) {
        handleApiError(error);
      } else {
        handleRetryableError(error);
      }
    },
  });

  const lookupStatus = isAxiosError(currentUser.error)
    ? currentUser.error.response?.status
    : undefined;
  const signedOut =
    currentUser.isError && (lookupStatus === 401 || lookupStatus === 403);
  const lookupFailed = currentUser.isError && !signedOut;
  const warning = currentUser.isSuccess
    ? accountSwitchMessage(currentUser.data.email)
    : null;

  if (errors.length > 0) {
    return (
      <AuthLayout>
        <div className="flex flex-col gap-3 text-center lg:text-left">
          <div className="flex justify-center pb-3 lg:hidden">
            <HatchetLogo className="h-8 w-auto" />
          </div>
          <h2 className="text-2xl font-semibold tracking-tight">
            This link can't be used
          </h2>
          <p className="text-sm text-muted-foreground">
            Login links expire after a short time and work only once.
          </p>
        </div>
        <Alert variant="destructive">
          <AlertDescription>{errors.join(' ')}</AlertDescription>
        </Alert>
        <Button asChild>
          <Link to={appRoutes.authLoginRoute.to}>Request a new link</Link>
        </Button>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout>
      <div className="flex flex-col gap-3 text-center lg:text-left">
        <div className="flex justify-center pb-3 lg:hidden">
          <HatchetLogo className="h-8 w-auto" />
        </div>
        <h2 className="text-2xl font-semibold tracking-tight">
          Confirm sign-in
        </h2>
        <p className="text-sm text-muted-foreground">
          This link can be used once. Continue only if you just requested it.
        </p>
      </div>
      {warning && (
        <Alert>
          <AlertDescription>{warning}</AlertDescription>
        </Alert>
      )}
      {lookupFailed && (
        <Alert variant="destructive">
          <AlertDescription className="flex flex-col gap-2">
            We couldn't check whether this browser is already signed in.
            <Button
              variant="outline"
              size="sm"
              className="self-start"
              disabled={currentUser.isFetching}
              onClick={() => currentUser.refetch()}
            >
              {currentUser.isFetching && <Spinner />}
              Try again
            </Button>
          </AlertDescription>
        </Alert>
      )}
      {retryErrors.length > 0 && (
        <Alert variant="destructive">
          <AlertDescription>
            {retryErrors.join(' ')} You can try again.
          </AlertDescription>
        </Alert>
      )}
      <Button
        disabled={
          !token || confirm.isPending || currentUser.isPending || lookupFailed
        }
        onClick={() => {
          if (!token) {
            return;
          }
          confirm.mutate({ token });
        }}
      >
        {(confirm.isPending || currentUser.isPending) && <Spinner />}
        Sign in
      </Button>
      <AuthLegalText />
    </AuthLayout>
  );
}
