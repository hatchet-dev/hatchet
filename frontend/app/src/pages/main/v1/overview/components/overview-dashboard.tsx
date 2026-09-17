import { ErrorsPanel } from './dashboard/errors-panel';
import { StatsPanel } from './dashboard/stats-panel';
import { TasksPanel } from './dashboard/tasks-panel';
import { WorkersPanel } from './dashboard/workers-panel';
import { openOnboarding } from './onboarding-modal';
import { SupportSection } from './support-section';
import { Button } from '@/components/v1/ui/button';
import { useMemo, useState } from 'react';

// The new, flag-gated Overview. Replaces the legacy onboarding-wizard-centric
// content with a live dashboard of tenant data. Composes the title, the
// onboarding re-entry highlight (until the tenant is onboarded), the panel
// grid, the Support footer, and the footer re-entry link. The onboarding modal
// itself is mounted globally, so the banner and footer only call openOnboarding.
export function OverviewDashboard({
  tenantId,
  onboarded,
}: {
  tenantId: string;
  onboarded: boolean;
}) {
  const [bannerDismissed, setBannerDismissed] = useState(false);
  const showBanner = !onboarded && !bannerDismissed;

  // A single 24h window shared by every panel that filters on it, so their
  // React Query keys match and the underlying requests are deduped.
  const since = useMemo(
    () => new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    [],
  );

  return (
    <div className="flex h-full w-full flex-col gap-y-6 lg:p-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
        <p className="text-sm text-muted-foreground">
          See activity across your Hatchet tenant.
        </p>
      </div>

      {showBanner && (
        <div className="flex flex-wrap items-start justify-between gap-4 rounded-lg border border-brand/40 bg-brand/5 p-4">
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
              Resume onboarding
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

      <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-2">
        <StatsPanel tenantId={tenantId} since={since} />
        <TasksPanel tenantId={tenantId} since={since} />
        <ErrorsPanel tenantId={tenantId} since={since} />
        <WorkersPanel tenantId={tenantId} />
      </div>

      <SupportSection />

      <div className="flex flex-wrap items-center gap-2 border-t border-border/50 pt-4 text-sm text-muted-foreground">
        <span>Need the setup guide again?</span>
        <Button
          variant="link"
          size="sm"
          className="h-auto p-0"
          onClick={openOnboarding}
        >
          Open onboarding
        </Button>
      </div>
    </div>
  );
}
