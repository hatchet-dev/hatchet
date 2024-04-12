import { Link, useNavigate } from 'react-router-dom';
import { UserRegisterForm } from './components/user-register-form';
import { cn } from '@/lib/utils';
import { buttonVariants } from '@/components/ui/button';
import { useMutation } from '@tanstack/react-query';
import api, { UserRegisterRequest } from '@/lib/api';
import { useState } from 'react';
import { useApiError } from '@/lib/hooks';
import useApiMeta from '../hooks/use-api-meta';
import { Loading } from '@/components/ui/loading';
import { GithubLogin, GoogleLogin, OrContinueWith } from '../login';
import useErrorParam from '../hooks/use-error-param';
import React from 'react';

export default function Register() {
  useErrorParam();
  const meta = useApiMeta();

  if (meta.isLoading) {
    return <Loading />;
  }

  const schemes = meta.data?.data?.auth?.schemes || [];
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
    <div className="flex flex-row flex-1 w-full h-full">
      <div className="container relative hidden flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
        <Link
          to="/auth/login"
          className={cn(
            buttonVariants({ variant: 'ghost' }),
            'absolute right-4 top-4 md:right-8 md:top-8',
          )}
        >
          Login
        </Link>
        <div className="lg:p-8 mx-auto w-screen">
          <div className="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[350px]">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">
                Create an account
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
      navigate('/');
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
