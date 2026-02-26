import { AuthPage } from '../components/auth-page';
import { UserLoginForm } from './components/user-login-form';
import api, { UserLoginRequest } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';

export default function Login() {
  return (
    <AuthPage
      title="Log in to continue"
      basicSection={<BasicLogin />}
      altAction={
        <>
          Don&apos;t have an account?{' '}
          <Link
            to={appRoutes.authRegisterRoute.to}
            className="font-semibold text-primary underline underline-offset-4 hover:text-primary/90"
          >
            Sign up
          </Link>
        </>
      }
    />
  );
}

function BasicLogin() {
  const navigate = useNavigate();
  const [errors, setErrors] = useState<string[]>([]);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({ setFieldErrors, setErrors });
  const { invalidate: invalidateUserUniverse, get: getUserUniverse } =
    useUserUniverse();

  const loginMutation = useMutation({
    mutationKey: ['user:update:login'],
    mutationFn: async (data: UserLoginRequest) => {
      await api.userUpdateLogin(data);
    },
    onSuccess: () => {
      invalidateUserUniverse();
      getUserUniverse();
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
