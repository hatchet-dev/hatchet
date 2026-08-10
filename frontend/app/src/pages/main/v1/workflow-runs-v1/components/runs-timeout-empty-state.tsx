import { RunsEmptyGraphic } from './runs-empty-graphic';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';

interface RequestTimeoutCloudCTAEmptyStateProps {
  utmCampaign: string;
}

export function RequestTimeoutCloudCTAEmptyState({
  utmCampaign,
}: RequestTimeoutCloudCTAEmptyStateProps) {
  const href = import.meta.env.DEV
    ? 'https://cloud.hatchet.run'
    : `https://cloud.hatchet.run?utm_source=timeout_cta&utm_medium=app&utm_campaign=${encodeURIComponent(utmCampaign)}`;

  return (
    <EmptyState
      graphic={<RunsEmptyGraphic />}
      title="This request timed out"
      description="Self-hosted instances can be slow to query at scale. Try our managed cloud for a faster, fully-managed experience."
      links={[
        {
          label: 'Try our managed cloud',
          href,
          external: true,
        },
      ]}
    />
  );
}
