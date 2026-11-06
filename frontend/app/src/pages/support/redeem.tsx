import { SetupCard, SetupScreen } from '@/components/layout/setup-card';
import { Spinner } from '@/components/v1/ui/loading';
import { controlPlaneApi } from '@/lib/api/api';
import { setSupportSession } from '@/lib/support-session';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { redirect } from '@tanstack/react-router';

/**
 * Runs in the route loader so the code is redeemed exactly once: an effect runs
 * twice under strict mode and would burn it.
 */
export async function loader({ code }: { code: string }) {
  const session = (await controlPlaneApi.supportSessionRedeem({ code })).data;

  setSupportSession(session);
  // Anything fetched before belongs to a different identity and backend.
  queryClient.clear();

  throw redirect({
    to: appRoutes.tenantRunsRoute.to,
    params: { tenant: session.grant.tenantId },
    replace: true,
  });
}

export function SupportRedeemPending() {
  return (
    <SetupScreen>
      <SetupCard title="Opening support session">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Spinner />
          Redeeming your support link…
        </div>
      </SetupCard>
    </SetupScreen>
  );
}

export function SupportRedeemError() {
  return (
    <SetupScreen>
      <SetupCard title="Support link unavailable">
        <p className="text-sm text-muted-foreground">
          This support link is invalid or has expired. Run{' '}
          <code className="font-mono">hat support</code> again to open the
          tenant.
        </p>
      </SetupCard>
    </SetupScreen>
  );
}

export default function SupportRedeem() {
  return <SupportRedeemPending />;
}
