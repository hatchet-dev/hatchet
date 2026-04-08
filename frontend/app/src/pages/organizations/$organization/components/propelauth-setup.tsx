import SsoSetup from '@/components/sso/sso-setup.tsx';
import type { SsoApi } from '@/lib/sso/sso-types.ts';
import { appRoutes } from '@/router.tsx';
import { useParams } from '@tanstack/react-router';

interface OrgSsoApi extends SsoApi {
  orgId: string;
}
const api: OrgSsoApi = {
  async get() {
    const r = await fetch(`/api/v1/management/organizations/${this.orgId}/sso`);
    if (!r.ok) {
      return {
        ok: false,
        error: { status: r.status, message: await r.text() },
      };
    }
    return { ok: true, data: await r.json() };
  },
  async upsert(body) {
    const r = await fetch(
      `/api/v1/management/organizations/${this.orgId}/sso`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      },
    );
    console.log(r);
    return r.ok
      ? { ok: true, data: undefined }
      : { ok: false, error: { status: r.status, message: await r.text() } };
  },
  async remove() {
    const r = await fetch(
      `/api/v1/management/organizations/${this.orgId}/sso`,
      {
        method: 'DELETE',
    });
    return r.ok
      ? { ok: true, data: undefined }
      : { ok: false, error: { status: r.status, message: await r.text() } };
  },
  orgId: '',
};

export default function PropelAuthPage() {
  const { organization: orgId } = useParams({
    from: appRoutes.organizationsRoute.to,
  });
  console.log(orgId);
  api.orgId = orgId;
  return (
    <SsoSetup
      redirectUrl="https://app.dev.hatchet-tools.com/auth/login"
      api={api}
    />
  );
}
