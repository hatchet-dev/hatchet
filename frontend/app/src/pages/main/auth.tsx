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

const authMiddleware = async (currentUrl: string) => {
  try {
    const user = await queryClient.fetchQuery({
      queryKey: ['user:get:current'],
      queryFn: async () => {
        const res = await api.userGetCurrent();

        return res.data;
      },
    });

    if (!user.emailVerified && !currentUrl.includes('/verify-email')) {
      throw redirect('/verify-email');
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

const membershipsPopulator = async (currentUrl: string) => {
  const res = await api.tenantMembershipsList();

  const memberships = res.data;

  if (memberships.rows?.length === 0 && !currentUrl.includes('/onboarding')) {
    throw redirect('/onboarding/create-tenant');
  }

  return res.data.rows;
};

export async function loader({ request }: LoaderFunctionArgs) {
  const user = await authMiddleware(request.url);
  const memberships = await membershipsPopulator(request.url);
  return {
    user,
    memberships,
  };
}

export default function Auth() {
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

  if (!user || !memberships) {
    return <Loading />;
  }

  return <Outlet context={ctx} />;
}
