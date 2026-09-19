import { ErrorsPanel } from './dashboard/errors-panel';
import { StatsPanel } from './dashboard/stats-panel';
import { TasksPanel } from './dashboard/tasks-panel';
import { WorkersPanel } from './dashboard/workers-panel';
import { SupportSection } from './support-section';
import { useTenantOnboarded } from './use-onboarding-progress';
import { useOpenOnboarding } from './use-open-onboarding';
import { Button } from '@/components/v1/ui/button';
import { useEffect, useState } from 'react';

const DAY_MS = 24 * 60 * 60 * 1000;
const WINDOW_REFRESH_MS = 5 * 60_000;

const dayAgo = () => new Date(Date.now() - DAY_MS).toISOString();

export function OverviewDashboard({ tenantId }: { tenantId: string }) {
  const openOnboarding = useOpenOnboarding(tenantId);
  const tenantSetup = useTenantOnboarded(tenantId);
  const [bannerDismissed, setBannerDismissed] = useState(false);
  // While the setup state is unknown the banner stays hidden, so it never
  // flashes at tenants that turn out to be set up.
  const showBanner =
    !tenantSetup.isLoading && !tenantSetup.onboarded && !bannerDismissed;

  // One 24h cutoff shared by every panel, so their React Query keys match and
  // the requests are deduped. It advances periodically: a cutoff fixed at mount
  // would silently widen the "last 24 hours" window on a tab left open.
  const [since, setSince] = useState(dayAgo);
  useEffect(() => {
    const id = window.setInterval(() => setSince(dayAgo()), WINDOW_REFRESH_MS);
    return () => window.clearInterval(id);
  }, []);

  return (
    <div className="flex h-full w-full flex-col gap-y-6 lg:p-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
        <p className="text-sm text-muted-foreground">
          See activity across your Hatchet tenant.
        </p>
      </div>

      {showBanner && (
        <div className="flex flex-wrap items-center justify-between gap-4 rounded-lg border border-brand/40 bg-brand/5 p-4">
          <div className="space-y-1">
            <p className="text-sm font-medium">Finish setting up Hatchet</p>
            <p className="text-sm text-muted-foreground">
              Connect your coding agent (or onboard manually), then run your
              first task. It takes about five minutes.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              className="border-brand text-brand hover:bg-brand/10 hover:text-brand"
              onClick={openOnboarding}
            >
              Run your first task
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="text-muted-foreground"
              onClick={() => setBannerDismissed(true)}
            >
              Dismiss for now
            </Button>
          </div>
        </div>
      )}

      {/* Stats spans the full width. Below it, two independent columns so each
          packs top-to-bottom on its own: Workers sits directly under Runs on
          the left regardless of how tall the Errors panel grows on the right. A
          single grid row would tie the two columns to a shared (Errors-driven)
          height and leave a large gap under Runs. */}
      <StatsPanel tenantId={tenantId} since={since} />
      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-2">
        <div className="flex flex-col gap-4">
          <TasksPanel tenantId={tenantId} since={since} />
          <WorkersPanel tenantId={tenantId} />
        </div>
        <div className="flex flex-col gap-4">
          <ErrorsPanel tenantId={tenantId} since={since} />
        </div>
      </div>

      <SupportSection />

      <div className="flex flex-wrap items-center gap-2 border-t border-border/50 pt-4 text-sm text-muted-foreground">
        <span>Want to run your first task?</span>
        <Button
          variant="link"
          size="sm"
          className="h-auto p-0"
          onClick={openOnboarding}
        >
          Run your first task
        </Button>
      </div>
    </div>
  );
}
