import { AuthLayout } from '../components/auth-layout';
import { AuthLegalText } from '../components/auth-legal-text';
import useApiMeta from '../hooks/use-api-meta';
import useErrorParam from '../hooks/use-error-param';
import { UserLoginForm } from './components/user-login-form';
import { Button } from '@/components/v1/ui/button';
import { Icons } from '@/components/v1/ui/icons';
import { Loading } from '@/components/v1/ui/loading';
import api, { UserLoginRequest } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import React from 'react';

export default function Login() {
  useErrorParam();
  const meta = useApiMeta();

  if (meta.isLoading) {
    return <Loading />;
  }

  const schemes = meta.data?.auth?.schemes || [];
  const basicEnabled = schemes.includes('basic');
  const googleEnabled = schemes.includes('google');
  const githubEnabled = schemes.includes('github');

  let prompt = 'Enter your email and password below.';

  if (basicEnabled && (googleEnabled || githubEnabled)) {
    prompt =
      'Enter your email and password below, or continue with a supported provider.';
  } else if (googleEnabled || githubEnabled) {
    prompt = 'Continue with a supported provider.';
  } else if (basicEnabled) {
    prompt = 'Enter your email and password below.';
  } else {
    prompt = 'No login methods are enabled.';
  }

  const forms = [
    basicEnabled && <BasicLogin />,
    googleEnabled && <GoogleLogin />,
    githubEnabled && <GithubLogin />,
  ].filter(Boolean);

  return (
    <AuthLayout>
      <div className="flex flex-col space-y-2 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">
          Log in to Hatchet
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
          Don't have an account?{' '}
          <Link
            to={appRoutes.authRegisterRoute.to}
            className="underline underline-offset-4 hover:text-primary"
          >
            Sign up
          </Link>
        </p>
      </div>
      <AuthLegalText />
    </AuthLayout>
  );
}

export function OrContinueWith() {
  return (
    <div className="relative my-4">
      <div className="absolute inset-0 flex items-center">
        <span className="w-full border-t" />
      </div>
      <div className="relative flex justify-center text-xs uppercase">
        <span className="bg-white px-2 text-gray-700 dark:bg-gray-800 dark:text-gray-300">
          Or continue with
        </span>
      </div>
    </div>
  );
}

function BasicLogin() {
  const navigate = useNavigate();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({ setFieldErrors });

  const loginMutation = useMutation({
    mutationKey: ['user:update:login'],
    mutationFn: async (data: UserLoginRequest) => {
      await api.userUpdateLogin(data);
    },
    onSuccess: () => {
      navigate({ to: appRoutes.authenticatedRoute.to });
    },
    onError: handleApiError,
  });

  return (
    <UserLoginForm
      isLoading={loginMutation.isPending}
      onSubmit={loginMutation.mutate}
      fieldErrors={fieldErrors}
    />
  );
}

export function GoogleLogin() {
  return (
    <a href="/api/v1/users/google/start" className="w-full">
      <Button
        variant="outline"
        type="button"
        fullWidth
        leftIcon={<Icons.google className="size-4" />}
      >
        Google
      </Button>
    </a>
  );
}

export function GithubLogin() {
  return (
    <a href="/api/v1/users/github/start" className="w-full">
      <Button
        variant="outline"
        type="button"
        fullWidth
        leftIcon={<Icons.gitHub className="size-4" />}
      >
        Github
      </Button>
    </a>
  );
}
