import MainNav from '@/components/molecules/nav-bar/nav-bar';
import {
  LoaderFunctionArgs,
  Outlet,
  redirect,
  useLoaderData,
} from 'react-router-dom';
import api, { queries } from '@/lib/api';
import queryClient from '@/query-client';
import { useContextFromParent } from '@/lib/outlet';
import { Loading } from '@/components/ui/loading.tsx';
import { useQuery } from '@tanstack/react-query';
import SupportChat from '@/components/molecules/support-chat';
import AnalyticsProvider from '@/components/molecules/analytics-provider';
import { useState } from 'react';

const authMiddleware = async (currentUrl: string) => {
  try {
    const user = await queryClient.fetchQuery({
      queryKey: ['user:get:current'],
      queryFn: async () => {
        const res = await api.userGetCurrent();

        return res.data;
      },
    });

    if (
      !user.emailVerified &&
      !currentUrl.includes('/onboarding/verify-email')
    ) {
      throw redirect('/onboarding/verify-email');
    }

    return user;
  } catch (error) {
    if (error instanceof Response) {
      throw error;
    } else if (
      !currentUrl.includes('/auth/login') &&
      !currentUrl.includes('/auth/register')
    ) {
      throw redirect('/auth/login');
    }
  }
};

const invitesRedirector = async (currentUrl: string) => {
  try {
    const res = await api.userListTenantInvites();

    const invites = res.data.rows || [];

    if (invites.length > 0 && !currentUrl.includes('/onboarding/invites')) {
      throw redirect('/onboarding/invites');
    }

    return;
  } catch (error) {
    if (error instanceof Response) {
      throw error;
    }
  }
};

const membershipsPopulator = async (currentUrl: string) => {
  try {
    const res = await api.tenantMembershipsList();

    const memberships = res.data;

    if (memberships.rows?.length === 0 && !currentUrl.includes('/onboarding')) {
      throw redirect('/onboarding/create-tenant');
    }

    return res.data.rows;
  } catch (error) {
    if (error instanceof Response) {
      throw error;
    }
  }
};

export async function loader({ request }: LoaderFunctionArgs) {
  const user = await authMiddleware(request.url);
  await invitesRedirector(request.url);
  const memberships = await membershipsPopulator(request.url);
  return {
    user,
    memberships,
  };
}

export default function Authenticated() {
  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  const { user, memberships } = useLoaderData() as Awaited<
    ReturnType<typeof loader>
  >;

  const ctx = useContextFromParent({
    user,
    memberships: listMembershipsQuery.data?.rows || memberships,
  });

  const [hasHasBanner, setHasBanner] = useState(false);

  if (!user || !memberships) {
    return <Loading />;
  }

  return (
    <AnalyticsProvider user={user}>
      <SupportChat user={user}>
        <div className="flex flex-row flex-1 w-full h-full">
          <MainNav user={user} setHasBanner={setHasBanner} />
          <div
            className={`pt-${hasHasBanner ? 28 : 16} flex-grow overflow-y-auto overflow-x-hidden`}
          >
            <Outlet context={ctx} />
          </div>
        </div>
      </SupportChat>
    </AnalyticsProvider>
  );
}
