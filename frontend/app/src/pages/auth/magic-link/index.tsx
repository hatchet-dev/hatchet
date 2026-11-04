import { AuthLayout } from '../components/auth-layout';
import { AuthLegalText } from '../components/auth-legal-text';
import { accountSwitchMessage } from './magic-link';
import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { useUserApi } from '@/lib/api/user-wrapper';
import { useApiError } from '@/lib/hooks';
import { initialMagicLinkToken } from '@/lib/magic-link-bootstrap';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';

export default function MagicLinkConfirm() {
  const token = initialMagicLinkToken;
  const [errors, setErrors] = useState<string[]>(
    token ? [] : ['This login link is missing or invalid.'],
  );
  const { handleApiError } = useApiError({ setErrors });
  const { invalidate: invalidateUserUniverse } = useUserUniverse();
  const navigate = useNavigate();
  const { userGetCurrentQuery, userUpdateMagicLinkMutation } = useUserApi();
  const currentUser = useQuery({
    ...userGetCurrentQuery(),
    retry: false,
  });
  const confirm = useMutation({
    ...userUpdateMagicLinkMutation(),
    onSuccess: async () => {
      await invalidateUserUniverse();
      await new Promise((resolve) => setTimeout(resolve, 0));
      navigate({ to: appRoutes.authenticatedRoute.to });
    },
    onError: handleApiError,
  });
  const warning = currentUser.isSuccess
    ? accountSwitchMessage(currentUser.data.email)
    : null;

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
      {errors.length > 0 && (
        <Alert variant="destructive">
          <AlertDescription>{errors.join(' ')}</AlertDescription>
        </Alert>
      )}
      <Button
        disabled={!token || confirm.isPending}
        onClick={() => {
          if (!token) {
            return;
          }
          setErrors([]);
          confirm.mutate({ token });
        }}
      >
        {confirm.isPending && <Spinner />}
        Sign in
      </Button>
      <div className="text-sm text-muted-foreground">
        Need a new link?{' '}
        <Link
          to={appRoutes.authLoginRoute.to}
          className="font-semibold text-primary underline underline-offset-4 hover:text-primary/90"
        >
          Request another one
        </Link>
      </div>
      <AuthLegalText />
    </AuthLayout>
  );
}
