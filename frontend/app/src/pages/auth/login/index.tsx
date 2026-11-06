import { AuthPage } from '../components/auth-page';
import { OrContinueWith } from '../components/social-auth';
import useApiMeta from '../hooks/use-api-meta';
import { showsMagicLinkLogin } from '../magic-link/magic-link';
import { MagicLinkLogin } from './components/magic-link-login';
import { UserLoginForm } from './components/user-login-form';
import { Button } from '@/components/v1/ui/button';
import useControlPlane from '@/hooks/use-control-plane';
import { useUserApi } from '@/lib/api/user-wrapper';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import { ArrowLeft, Mail } from 'lucide-react';
import { useState } from 'react';

export default function Login() {
  const { isControlPlaneEnabled } = useControlPlane();
  const { meta } = useApiMeta();
  const schemes = meta?.auth?.schemes ?? [];
  const showBasic = schemes.includes('basic');
  const showMagicLink = showsMagicLinkLogin(isControlPlaneEnabled, schemes);
  const [magicLinkExpanded, setMagicLinkExpanded] = useState(false);

  return (
    <AuthPage
      title="Log in to continue"
      showLocalAuth={showBasic || showMagicLink}
      basicSection={
        magicLinkExpanded ? (
          <div className="flex flex-col gap-2">
            <MagicLinkLogin />
            <Button
              variant="ghost"
              type="button"
              size="sm"
              onClick={() => setMagicLinkExpanded(false)}
              className="h-11 justify-left gap-1 border-muted-foreground/20 bg-background shadow-sm hover:bg-muted/40"
            >
              <ArrowLeft className="size-4" />
              Back
            </Button>
          </div>
        ) : (
          <>
            {showBasic && <BasicLogin />}
            {showBasic && showMagicLink && <OrContinueWith />}
            {showMagicLink && (
              <Button
                variant="outline"
                type="button"
                fullWidth
                onClick={() => setMagicLinkExpanded(true)}
                className="h-11 justify-center gap-2 border-muted-foreground/20 bg-background shadow-sm hover:bg-muted/40"
              >
                <Mail className="size-4" />
                Email me a login link
              </Button>
            )}
          </>
        )
      }
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
  const { invalidate: invalidateUserUniverse } = useUserUniverse();

  const { userUpdateLoginMutation } = useUserApi();
  const loginMutation = useMutation({
    ...userUpdateLoginMutation(),
    onSuccess: async () => {
      // A pending session is sent to /auth/mfa by the response interceptor when
      // the next control-plane call returns mfa_required.
      await invalidateUserUniverse();
      await new Promise((resolve) => setTimeout(resolve, 0));
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
