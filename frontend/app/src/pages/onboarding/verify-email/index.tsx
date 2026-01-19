import TopNav from '@/components/v1/nav/top-nav';
import { Loading } from '@/components/v1/ui/loading';
import { useAnalytics } from '@/hooks/use-analytics';
import { queries } from '@/lib/api';
import { AppContextProvider } from '@/providers/app-context';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { redirect, useLoaderData } from '@tanstack/react-router';
import { useEffect } from 'react';

export async function loader({ request }: { request: Request }) {
  try {
    const user = await queryClient.fetchQuery(queries.user.current);

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

  useEffect(() => {
    capture('onboarding_verify_email_viewed');
  }, [capture]);

  if (!res?.user) {
    return <Loading />;
  }

  return (
    <div className="flex h-full w-full flex-1 flex-col">
      <TopNav user={res.user} tenantMemberships={[]} />
      <div className="container relative hidden flex-1 flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0 -mt-48">
        <div className="mx-auto w-screen lg:p-8">
          <div className="mx-auto flex w-40 flex-col justify-center space-y-6 sm:w-[350px]">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">
                Verify your email
              </h1>
            </div>
            <div className="my-4 text-sm">
              Please contact your Hatchet instance administrator to verify your
              email. Refresh this page once your email has been verified.
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function VerifyEmail() {
  return (
    <AppContextProvider>
      <VerifyEmailInner />
    </AppContextProvider>
  );
}
