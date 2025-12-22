import { AuthLayout } from '../components/auth-layout';
import { AuthLegalText } from '../components/auth-legal-text';
import useApiMeta from '../hooks/use-api-meta';
import useErrorParam from '../hooks/use-error-param';
import { GithubLogin, GoogleLogin, OrContinueWith } from '../login';
import { UserRegisterForm } from './components/user-register-form';
import { Loading } from '@/components/v1/ui/loading';
import {
  POSTHOG_DISTINCT_ID_LOCAL_STORAGE_KEY,
  POSTHOG_SESSION_ID_LOCAL_STORAGE_KEY,
} from '@/hooks/use-analytics';
import api, { UserRegisterRequest } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import React from 'react';

export default function Register() {
  useErrorParam();
  const meta = useApiMeta();

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

  if (meta.isLoading) {
    return <Loading />;
  }

  const schemes = meta.data?.auth?.schemes || [];
  const basicEnabled = schemes.includes('basic');
  const googleEnabled = schemes.includes('google');
  const githubEnabled = schemes.includes('github');

  let prompt = 'Create an account to get started.';

  if (basicEnabled && (googleEnabled || githubEnabled)) {
    prompt =
      'Enter your email and password to create an account, or continue with a supported provider.';
  } else if (googleEnabled || githubEnabled) {
    prompt = 'Continue with a supported provider.';
  } else if (basicEnabled) {
    prompt = 'Create an account to get started.';
  } else {
    prompt = 'No login methods are enabled.';
  }

  const forms = [
    basicEnabled && <BasicRegister />,
    googleEnabled && <GoogleLogin />,
    githubEnabled && <GithubLogin />,
  ].filter(Boolean);

  return (
    <AuthLayout>
      <div className="flex flex-col space-y-2 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">
          Create an account
        </h1>
        <p className="text-sm text-gray-700 dark:text-gray-300">{prompt}</p>
      </div>
      {forms.map((form, index) => (
        <React.Fragment key={index}>
          {form}
          {index < schemes.length - 1 && <OrContinueWith />}
        </React.Fragment>
      ))}
      <div className="flex flex-col space-y-2">
        <p className="text-sm text-gray-700 dark:text-gray-300">
          Already have an account?{' '}
          <Link
            to={appRoutes.authLoginRoute.to}
            className="underline underline-offset-4 hover:text-primary"
          >
            Log in
          </Link>
        </p>
      </div>
      <AuthLegalText />
    </AuthLayout>
  );
}

function BasicRegister() {
  const navigate = useNavigate();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createMutation = useMutation({
    mutationKey: ['user:create'],
    mutationFn: async (data: UserRegisterRequest) => {
      await api.userCreate(data);
    },
    onSuccess: () => {
      navigate({ to: appRoutes.authenticatedRoute.to });
    },
    onError: handleApiError,
  });

  return (
    <UserRegisterForm
      isLoading={createMutation.isPending}
      onSubmit={createMutation.mutate}
      fieldErrors={fieldErrors}
    />
  );
}
