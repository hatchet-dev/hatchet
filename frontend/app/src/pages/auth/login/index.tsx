import { Link, useNavigate } from 'react-router-dom';
import { UserLoginForm } from './components/user-login-form';
import { cn } from '@/lib/utils';
import { Button, buttonVariants } from '@/components/ui/button';
import { useMutation } from '@tanstack/react-query';
import api, { UserLoginRequest } from '@/lib/api';
import { useState } from 'react';
import { useApiError } from '@/lib/hooks';
import useApiMeta from '../hooks/use-api-meta';
import { Loading } from '@/components/ui/loading';
import { Icons } from '@/components/ui/icons';
import useErrorParam from '../hooks/use-error-param';
import React from 'react';

export default function Login() {
  useErrorParam();
  const meta = useApiMeta();

  if (meta.isLoading) {
    return <Loading />;
  }

  const schemes = meta.data?.data?.auth?.schemes || [];
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
    <div className="flex flex-row flex-1 w-full h-full">
      <div className="container relative hidden flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
        <Link
          to="/auth/register"
          className={cn(
            buttonVariants({ variant: 'ghost' }),
            'absolute right-4 top-4 md:right-8 md:top-8',
          )}
        >
          Register
        </Link>
        <div className="lg:p-8 mx-auto w-screen">
          <div className="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[350px]">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">
                Log in to Hatchet
              </h1>
              <p className="text-sm text-muted-foreground">{prompt}</p>
            </div>
            {forms.map((form, index) => (
              <React.Fragment key={index}>
                {form}
                {index < schemes.length - 1 && <OrContinueWith />}
              </React.Fragment>
            ))}
            <p className="text-left text-sm text-muted-foreground w-full">
              By clicking continue, you agree to our{' '}
              <Link
                to="/terms"
                className="underline underline-offset-4 hover:text-primary"
              >
                Terms of Service
              </Link>{' '}
              and{' '}
              <Link
                to="/privacy"
                className="underline underline-offset-4 hover:text-primary"
              >
                Privacy Policy
              </Link>
              .
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

export function OrContinueWith() {
  return (
    <div className="relative">
      <div className="absolute inset-0 flex items-center">
        <span className="w-full border-t" />
      </div>
      <div className="relative flex justify-center text-xs uppercase">
        <span className="bg-background px-2 text-muted-foreground">
          Or continue with
        </span>
      </div>
    </div>
  );
}

function BasicLogin() {
  const navigate = useNavigate();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const loginMutation = useMutation({
    mutationKey: ['user:update:login'],
    mutationFn: async (data: UserLoginRequest) => {
      await api.userUpdateLogin(data);
    },
    onSuccess: () => {
      navigate('/');
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
    <a href="/api/v1/users/google/start">
      <Button variant="outline" type="button" className="w-full py-2">
        <Icons.google className="mr-2 h-4 w-4" />
        Google
      </Button>
    </a>
  );
}

export function GithubLogin() {
  return (
    <a href="/api/v1/users/github/start">
      <Button variant="outline" type="button" className="w-full py-2">
        <Icons.gitHub className="mr-2 h-4 w-4" />
        Github
      </Button>
    </a>
  );
}
