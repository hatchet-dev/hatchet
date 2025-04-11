import { Link, useNavigate } from 'react-router-dom';
import { UserLoginForm } from './components/user-login-form';
import React from 'react';
import useApiMeta from '@/next/hooks/use-api-meta';
import {
  AuthLayout,
  GoogleLogin,
  GithubLogin,
  OrContinueWith,
} from '../components/shared-auth-components';
import useUser from '@/next/hooks/use-user';

export default function Login() {
  const { oss: meta, isLoading } = useApiMeta();

  if (isLoading) {
    return 'Loading...'; // TODO: add loading
  }

  if (!meta) {
    return 'Error loading meta'; // TODO: add error
  }

  // const schemes = meta.auth?.schemes || [];
  const schemes = ['basic', 'google', 'github'];

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
    <AuthLayout title="Log in to Hatchet" prompt={prompt}>
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
            to="/auth/register"
            className="underline underline-offset-4 hover:text-primary"
          >
            Sign up
          </Link>
        </p>
      </div>
    </AuthLayout>
  );
}

function BasicLogin() {
  const navigate = useNavigate();
  const { login } = useUser();

  return (
    <UserLoginForm
      isLoading={login.isPending}
      onSubmit={async (data) => {
        const user = await login.mutateAsync(data);
        if (user) {
          navigate('/');
        }
      }}
      apiError={login.error?.message}
    />
  );
}
