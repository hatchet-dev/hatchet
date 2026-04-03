import api from '@/lib/api';
import { controlPlaneApi, fetchControlPlaneStatus } from '@/lib/api/api';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { redirect } from '@tanstack/react-router';
import { AxiosError, isAxiosError } from 'axios';

const noAuthMiddleware = async () => {
  try {
    const { isControlPlaneEnabled } = await fetchControlPlaneStatus();
    const user = await queryClient.fetchQuery({
      queryKey: ['user:get'],
      queryFn: async () =>
        (await (isControlPlaneEnabled
          ? controlPlaneApi.cloudUserGetCurrent()
          : api.userGetCurrent())
        ).data,
    });

    if (user) {
      throw redirect({ to: appRoutes.authenticatedRoute.to });
    }
  } catch (error) {
    if (error instanceof Response) {
      throw error;
    } else if (isAxiosError(error)) {
      const axiosErr = error as AxiosError;

      if (axiosErr.response?.status === 403) {
        return;
      } else {
        throw error;
      }
    }
  }
};

export async function loader() {
  await noAuthMiddleware();
  return null;
}
