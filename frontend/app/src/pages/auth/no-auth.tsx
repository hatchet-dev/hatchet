import api from '@/lib/api';
import { controlPlaneApi, fetchControlPlaneStatus } from '@/lib/api/api';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { redirect } from '@tanstack/react-router';

const noAuthMiddleware = async () => {
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

    if (user) {
      throw redirect({ to: appRoutes.authenticatedRoute.to });
    }
  } catch (error) {
    // Redirects must propagate. Anything else (logged out, aborted during
    // logout, or the session request failing) means stay on the login page.
    if (error instanceof Response) {
      throw error;
    }
  }
};

export async function loader() {
  await noAuthMiddleware();
  return null;
}
