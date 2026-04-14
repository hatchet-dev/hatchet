import SsoSetup from '@/components/sso/sso-setup.tsx';
import { appRoutes } from '@/router.tsx';
import { useParams } from '@tanstack/react-router';

export default function SSOPage() {
  const { organization: orgId } = useParams({
    from: appRoutes.organizationsRoute.to,
  });

  return (
    <SsoSetup
      redirectUrl={`${window.location.origin}/api/v1/cloud/users/sso/callback`}
      orgId={orgId}
    />
  );
}
