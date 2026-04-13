import SsoSetup from '@/components/sso/sso-setup.tsx';
import { cloudApi } from '@/lib/api/api';
import type { SsoApi } from '@/lib/sso/sso-types.ts';
import { appRoutes } from '@/router.tsx';
import { useParams } from '@tanstack/react-router';
import { AxiosError } from 'axios';

const makeApi = (orgId: string): SsoApi => ({
  async get() {
    try {
      const r = await cloudApi.ssoGet(orgId);
      return { ok: true, data: r.data };
    } catch (error: any) {
      if (error instanceof AxiosError) {
        return {
          ok: false,
          error: { status: error.status, message: error.message },
        };
      } else {
        return {
          ok: false,
          error: { message: 'unspecified error' },
        };
      }
    }
  },
  async upsert(body) {
    try {
      await cloudApi.ssoUpsert(orgId, body);
      return { ok: true, data: undefined };
    } catch (error: any) {
      if (error instanceof AxiosError) {
        return {
          ok: false,
          error: { status: error.status, message: error.message },
        };
      } else {
        return {
          ok: false,
          error: { message: 'unspecified error' },
        };
      }
    }
  },
  async remove() {
    try {
      await cloudApi.ssoDelete(orgId);
      return { ok: true, data: undefined };
    } catch (error: any) {
      if (error instanceof AxiosError) {
        return {
          ok: false,
          error: { status: error.status, message: error.message },
        };
      } else {
        return {
          ok: false,
          error: { message: 'unspecified error' },
        };
      }
    }
  },
});

export default function SSOPage() {
  const { organization: orgId } = useParams({
    from: appRoutes.organizationsRoute.to,
  });

  return (
    <SsoSetup
      redirectUrl={`${window.location.origin}/api/v1/cloud/users/sso/callback`}
      api={makeApi(orgId)}
    />
  );
}
