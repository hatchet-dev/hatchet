import { redirect } from '@tanstack/react-router';
import api from '@/lib/api';
import queryClient from '@/query-client';
import { AxiosError, isAxiosError } from 'axios';
import { appRoutes } from '@/router';

const noAuthMiddleware = async () => {
  try {
    const user = await queryClient.fetchQuery({
      queryKey: ['user:get:current'],
      queryFn: async () => {
        const res = await api.userGetCurrent();

        return res.data;
      },
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
