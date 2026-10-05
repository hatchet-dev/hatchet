import { OverviewDashboard } from './components/overview-dashboard';
import { useTenantDetails } from '@/hooks/use-tenant';

export default function Overview() {
  const { tenantId } = useTenantDetails();

  if (!tenantId) {
    return null;
  }

  // Keyed by tenant so a tenant switch remounts the dashboard: its panels keep
  // their previous data while refetching, which must never be another tenant's.
  return <OverviewDashboard key={tenantId} tenantId={tenantId} />;
}
