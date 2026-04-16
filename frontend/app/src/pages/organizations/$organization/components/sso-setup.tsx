import SsoSetup from '@/components/sso/sso-setup.tsx';
import { cloudApi, controlPlaneApi } from '@/lib/api/api';
import type { SsoApi } from '@/lib/sso/sso-types.ts';
import { appRoutes } from '@/router.tsx';
import { useParams } from '@tanstack/react-router';
import { AxiosError } from 'axios';
import { useMemo } from 'react';

function makeApi(orgId: string): SsoApi {
  return {
    async get() {
      try {
        const r = await controlPlaneApi.ssoGet(orgId);
        return { ok: true, data: r.data };
      } catch (error: any) {
        if (error instanceof AxiosError) {
          return {
            ok: false,
            error: { status: error.status, message: error.message },
          };
        }
        return { ok: false, error: { message: 'unspecified error' } };
      }
    },
    async upsert(body) {
      try {
        await controlPlaneApi.ssoUpsert(orgId, body);
        return { ok: true, data: undefined };
      } catch (error: any) {
        if (error instanceof AxiosError) {
          return {
            ok: false,
            error: { status: error.status, message: error.message },
          };
        }
        return { ok: false, error: { message: 'unspecified error' } };
      }
    },
    async remove() {
      try {
        await controlPlaneApi.ssoDelete(orgId);
        return { ok: true, data: undefined };
      } catch (error: any) {
        if (error instanceof AxiosError) {
          return {
            ok: false,
            error: { status: error.status, message: error.message },
          };
        }
        return { ok: false, error: { message: 'unspecified error' } };
      }
    },
  };
}

export default function SSOPage() {
  const { organization: orgId } = useParams({
    from: appRoutes.organizationsRoute.to,
  });

  const api = useMemo(() => makeApi(orgId), [orgId]);

  return (
    <SsoSetup
      redirectUrl={`${window.location.origin}/api/v1/cloud/users/sso/callback`}
      api={api}
    />
  );
}
