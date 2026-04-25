import SsoSetup from '@/components/sso/sso-setup.tsx';
import { controlPlaneApi } from '@/lib/api/api.ts';
import type { SsoApi } from '@/lib/sso/sso-types.ts';
import { AxiosError } from 'axios';
import { useMemo } from 'react';

function makeApi(orgId: string): SsoApi {
  return {
    async get() {
      try {
        const r = await controlPlaneApi.ssoList(orgId);
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
        await controlPlaneApi.ssoUpdate(orgId, body);
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

export default function CreateSSOPage({ orgId }: { orgId: string }) {
  const api = useMemo(() => makeApi(orgId), [orgId]);

  return (
    <SsoSetup
      redirectUrl={`${window.location.origin}/api/v1/control-plane/users/sso/callback`}
      api={api}
    />
  );
}
