import { Link } from 'react-router-dom';
import { UserLoginForm } from './components/user-login-form';
import { Loading } from '@/components/ui/loading';
import React, { useState } from 'react';
import useApiMeta from '@/next/hooks/use-api-meta';
import useErrorParam from '@/pages/auth/hooks/use-error-param';
import { ROUTES } from '@/next/lib/routes';
import useUser from '@/next/hooks/use-user';
import { AxiosError } from 'axios';
import {
  GithubLogin,
  GoogleLogin,
  OrContinueWith,
} from '../components/shared-auth-components';

export default function Login() {
  useErrorParam();
  const meta = useApiMeta();

  if (meta.isLoading) {
    return <Loading />;
  }

  const schemes = meta.oss?.auth?.schemes || [];
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
  ].filter((x) => x !== undefined);

  return (
    <div className="flex flex-1 flex-col items-center justify-center w-full h-full lg:flex-row">
      <div className="container relative flex-col items-center justify-center w-full lg:px-0">
        <div className="mx-auto flex w-full max-w-md lg:p-8">
          <div className="flex w-full flex-col justify-center space-y-6">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">
                Log in to Hatchet
              </h1>
              <p className="text-sm text-gray-700 dark:text-gray-300">
                {prompt}
              </p>
            </div>
            {forms.map((form, index) => (
              <React.Fragment key={index}>
                {form}
                {basicEnabled && schemes.length >= 2 && index === 0 ? (
                  <OrContinueWith />
                ) : null}
              </React.Fragment>
            ))}

            <div className="flex flex-col space-y-2">
              <p className="text-sm text-gray-700 dark:text-gray-300">
                Don't have an account?{' '}
                <Link
                  to={ROUTES.auth.register}
                  className="underline underline-offset-4 hover:text-primary"
                >
                  Sign up
                </Link>
              </p>
            </div>
            <p className="text-left text-sm text-gray-700 dark:text-gray-300 w-full">
              By clicking continue, you agree to our{' '}
              <Link
                to="https://www.iubenda.com/terms-and-conditions/76608149"
                className="underline underline-offset-4 hover:text-primary"
              >
                Terms of Service
              </Link>
              ,{' '}
              <Link
                to="https://www.iubenda.com/privacy-policy/76608149/cookie-policy"
                className="underline underline-offset-4 hover:text-primary"
              >
                Cookie Policy
              </Link>
              , and{' '}
              <Link
                to="https://www.iubenda.com/privacy-policy/76608149"
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

function BasicLogin() {
  const { login } = useUser();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  return (
    <UserLoginForm
      isLoading={login.isPending}
      onSubmit={async (data) => {
        try {
          await login.mutateAsync(data);
          window.location.href = '/';
        } catch (error) {
          if (error instanceof AxiosError) {
            setFieldErrors(error.response?.data.errors || {});
          }
        }
      }}
      fieldErrors={fieldErrors}
    />
  );
}
