import { AuthPage } from '../components/auth-page';
import { UserRegisterForm } from './components/user-register-form';
import {
  POSTHOG_DISTINCT_ID_LOCAL_STORAGE_KEY,
  POSTHOG_SESSION_ID_LOCAL_STORAGE_KEY,
} from '@/hooks/use-analytics';
import api, { UserRegisterRequest } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import { useEffect, useState } from 'react';

export default function Register() {
  // allows for cross-domain tracking with PostHog
  // see: https://posthog.com/tutorials/cross-domain-tracking
  // for setup instructions and more details
  // important: we need to set these in local storage from here,
  // because once we redirect after the user signs up, we lose the hash params
  const hashParams = new URLSearchParams(window.location.hash.substring(1));
  const distinctId = hashParams.get('distinct_id');
  const sessionId = hashParams.get('session_id');

  useEffect(() => {
    if (distinctId) {
      sessionStorage.setItem(POSTHOG_DISTINCT_ID_LOCAL_STORAGE_KEY, distinctId);
    }

    if (sessionId) {
      sessionStorage.setItem(POSTHOG_SESSION_ID_LOCAL_STORAGE_KEY, sessionId);
    }
  }, [distinctId, sessionId]);

  return (
    <AuthPage
      title="Create an account"
      basicSection={<BasicRegister />}
      altAction={
        <>
          Already have an account?{' '}
          <Link
            to={appRoutes.authLoginRoute.to}
            className="font-semibold text-primary underline underline-offset-4 hover:text-primary/90"
          >
            Log in
          </Link>
        </>
      }
    />
  );
}

function BasicRegister() {
  const navigate = useNavigate();
  const [errors, setErrors] = useState<string[]>([]);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { get: getUserUniverse } = useUserUniverse();
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
    setErrors: setErrors,
  });

  const createMutation = useMutation({
    mutationKey: ['user:create'],
    mutationFn: async (data: UserRegisterRequest) => {
      await api.userCreate(data);
    },
    onSuccess: () => {
      getUserUniverse();
      navigate({ to: appRoutes.authenticatedRoute.to });
    },
    onError: handleApiError,
  });

  return (
    <UserRegisterForm
      isLoading={createMutation.isPending}
      onSubmit={(data) => {
        setErrors([]);
        setFieldErrors({});
        createMutation.mutate(data);
      }}
      errors={errors}
      fieldErrors={fieldErrors}
    />
  );
}
