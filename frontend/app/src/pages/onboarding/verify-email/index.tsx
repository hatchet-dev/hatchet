import api from '@/lib/api';
import { LoaderFunctionArgs, redirect, useLoaderData } from 'react-router-dom';
import queryClient from '@/query-client';
import MainNav from '@/components/molecules/nav-bar/nav-bar';
import { Loading } from '@/components/ui/loading';

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
    <div className="flex flex-row flex-1 w-full h-full pt-16">
      <MainNav user={res.user} />
      <div className="container relative hidden flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
        <div className="lg:p-8 mx-auto w-screen">
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
