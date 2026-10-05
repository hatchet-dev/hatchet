import { SetupCard, SetupScreen } from '@/components/layout/setup-card';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading';
import { useAnalytics } from '@/hooks/use-analytics';
import api from '@/lib/api';
import { controlPlaneApi, fetchControlPlaneStatus } from '@/lib/api/api';
import { useUserApi } from '@/lib/api/user-wrapper';
import { AppContextProvider } from '@/providers/app-context';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { redirect, useLoaderData, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';

export async function loader({ request }: { request: Request }) {
  try {
    const { isControlPlaneEnabled } = await fetchControlPlaneStatus();
    const user = await queryClient.fetchQuery({
      queryKey: ['user:get'],
      queryFn: async () =>
        (
          await (isControlPlaneEnabled
            ? controlPlaneApi.cloudUserGetCurrent()
            : api.userGetCurrent())
        ).data,
    });

    if (
      user.emailVerified &&
      request.url.includes('/onboarding/verify-email')
    ) {
      throw redirect({ to: appRoutes.authenticatedRoute.to });
    }

    return { user };
  } catch (error) {
    if (error instanceof Response) {
      throw error;
    } else if (
      !request.url.includes('/auth/login') &&
      !request.url.includes('/auth/register')
    ) {
      throw redirect({ to: appRoutes.authLoginRoute.to });
    }
  }
}

function VerifyEmailInner() {
  const res = useLoaderData({
    from: appRoutes.onboardingVerifyRoute.to,
  }) as Awaited<ReturnType<typeof loader>>;
  const { capture } = useAnalytics();
  const navigate = useNavigate();
  const { userUpdateLogoutMutation } = useUserApi();

  // The full-screen card replaces the top nav (and its account menu), so the
  // page carries its own sign-out, matching the other setup screens.
  const logoutMutation = useMutation({
    ...userUpdateLogoutMutation(),
    onSettled: () => {
      queryClient.clear();
      navigate({ to: appRoutes.authLoginRoute.to });
    },
  });

  useEffect(() => {
    capture('onboarding_verify_email_viewed');
  }, [capture]);

  if (!res?.user) {
    return <Loading />;
  }

  return (
    <SetupScreen
      topRight={
        <Button
          variant="ghost"
          size="sm"
          onClick={() => logoutMutation.mutate()}
          disabled={logoutMutation.isPending}
        >
          Sign out
        </Button>
      }
    >
      <SetupCard
        title="Verify your email"
        description={`Signed in as ${res.user.email}.`}
        footer={
          <Button size="sm" onClick={() => window.location.reload()}>
            Refresh
          </Button>
        }
      >
        <p className="text-sm text-muted-foreground">
          Please contact your Hatchet instance administrator to verify your
          email. Refresh this page once your email has been verified.
        </p>
      </SetupCard>
    </SetupScreen>
  );
}

export default function VerifyEmail() {
  return (
    <AppContextProvider>
      <VerifyEmailInner />
    </AppContextProvider>
  );
}
