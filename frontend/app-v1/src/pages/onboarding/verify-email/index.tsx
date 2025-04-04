import api from '@/lib/api';
import { LoaderFunctionArgs, redirect, useLoaderData } from 'react-router-dom';
import queryClient from '@/query-client';
import MainNav from '@/components/molecules/nav-bar/nav-bar';
import { Loading } from '@/components/ui/loading';
import { AuthLayout } from '../../../../app/src/pages/auth/auth.layout';

export async function loader({ request }: LoaderFunctionArgs) {
  try {
    const user = await queryClient.fetchQuery({
      queryKey: ['user:get:current'],
      queryFn: async () => {
        const res = await api.userGetCurrent();

        return res.data;
      },
    });

    if (
      user.emailVerified &&
      request.url.includes('/onboarding/verify-email')
    ) {
      throw redirect('/');
    }

    return { user };
  } catch (error) {
    if (error instanceof Response) {
      throw error;
    } else if (
      !request.url.includes('/auth/login') &&
      !request.url.includes('/auth/register')
    ) {
      throw redirect('/auth/login');
    }
  }
}

export default function VerifyEmail() {
  const res = useLoaderData() as Awaited<ReturnType<typeof loader>>;

  if (!res?.user) {
    return <Loading />;
  }

  return (
    <div className="flex flex-row flex-1 w-full h-full">
      <MainNav user={res.user} />
      <AuthLayout
        title="Verify your email"
        prompt="Please contact your Hatchet instance administrator to verify your email. Refresh this page once your email has been verified."
      />
    </div>
  );
}
