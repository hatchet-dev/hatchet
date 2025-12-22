import { AuthPage } from '../components/auth-page';
import { UserLoginForm } from './components/user-login-form';
import api, { UserLoginRequest } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';

export default function Login() {
  return (
    <AuthPage
      title="Log in to Hatchet"
      promptFn={({ basicEnabled, googleEnabled, githubEnabled }) => {
        if (basicEnabled && (googleEnabled || githubEnabled)) {
          return 'Enter your email and password below, or continue with a supported provider.';
        }

        if (googleEnabled || githubEnabled) {
          return 'Continue with a supported provider.';
        }

        if (basicEnabled) {
          return 'Enter your email and password below.';
        }

        return 'No login methods are enabled.';
      }}
      basicSection={<BasicLogin />}
      footer={
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
      }
    />
  );
}

function BasicLogin() {
  const navigate = useNavigate();
  const [errors, setErrors] = useState<string[]>([]);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({ setFieldErrors, setErrors });

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
      onSubmit={(data) => {
        setErrors([]);
        setFieldErrors({});
        loginMutation.mutate(data);
      }}
      errors={errors}
      fieldErrors={fieldErrors}
    />
  );
}
