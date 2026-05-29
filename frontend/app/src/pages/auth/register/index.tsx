import { AuthPage } from '../components/auth-page';
import { UserRegisterForm } from './components/user-register-form';
import { useUserApi } from '@/lib/api/user-wrapper';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';

export default function Register() {
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

  const { userCreateMutation } = useUserApi();
  const createMutation = useMutation({
    ...userCreateMutation(),
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
