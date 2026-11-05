import { SettingsPageHeader } from '../components/settings-page-header';
import { SettingRow } from '../components/settings-row';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { Button } from '@/components/v1/ui/button';
import { Loading, Spinner } from '@/components/v1/ui/loading';
import { Switch } from '@/components/v1/ui/switch';
import { useState } from 'react';

export type LabsFlag =
  'durableStreams' | 'sandboxOperator' | 'serverlessOperator';

const labsFeatures: {
  key: LabsFlag;
  label: string;
  description: string;
  disableDescription: string;
}[] = [
  {
    key: 'durableStreams',
    label: 'Durable streams',
    description: 'Publish and subscribe to persistent topics from tasks.',
    disableDescription:
      'Tasks can no longer publish or subscribe. Existing topics are kept.',
  },
  {
    key: 'sandboxOperator',
    label: 'Sandboxes',
    description: 'Run untrusted code in isolated environments.',
    disableDescription: 'Sandboxes will be unavailable for this tenant.',
  },
  {
    key: 'serverlessOperator',
    label: 'Serverless',
    description: 'Run tasks on platforms like Vercel and Cloudflare Workers.',
    disableDescription:
      'Endpoints stop receiving tasks. Existing endpoints are kept.',
  },
];

export type LabsPreviewProps =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | {
      status: 'upgrade';
      canUpgrade: boolean;
      upgradePending: boolean;
      error?: string;
      onUpgrade: () => void;
      onViewBilling: () => void;
    }
  | { status: 'gated' }
  | {
      status: 'ready';
      flags: Record<LabsFlag, boolean>;
      pending: boolean;
      error?: string;
      onToggle: (flag: LabsFlag, enabled: boolean) => void;
    };

export function LabsPreview(props: LabsPreviewProps) {
  if (props.status === 'loading') {
    return <Loading />;
  }

  if (props.status === 'gated') {
    return (
      <div className="flex min-h-96 flex-1 items-center justify-center p-4">
        <div className="max-w-lg space-y-3 text-center">
          <h3 className="text-lg font-semibold">Owner access required</h3>
          <p className="text-sm text-muted-foreground">
            Ask a tenant owner to change Labs settings.
          </p>
        </div>
      </div>
    );
  }

  if (props.status === 'error') {
    return (
      <div className="flex min-h-96 flex-1 items-center justify-center p-4">
        <p className="text-sm text-destructive">{props.message}</p>
      </div>
    );
  }

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Labs Preview"
          description="Early access features. These may change before general availability."
        />

        {props.status === 'upgrade' ? (
          <UpgradeSurface {...props} />
        ) : (
          <LabsSwitches {...props} />
        )}
      </div>
    </div>
  );
}

function LabsSwitches({
  flags,
  pending,
  error,
  onToggle,
}: Extract<LabsPreviewProps, { status: 'ready' }>) {
  const [disabling, setDisabling] = useState<LabsFlag | null>(null);
  const disablingFeature = labsFeatures.find(
    (feature) => feature.key === disabling,
  );

  return (
    <div>
      <div className="divide-y divide-border">
        {labsFeatures.map((feature) => (
          <SettingRow
            key={feature.key}
            label={feature.label}
            description={feature.description}
          >
            <Switch
              checked={flags[feature.key]}
              disabled={pending}
              aria-label={feature.label}
              onCheckedChange={(enabled) => {
                if (enabled) {
                  onToggle(feature.key, true);
                  return;
                }

                setDisabling(feature.key);
              }}
            />
          </SettingRow>
        ))}
      </div>
      {error ? <p className="pt-4 text-sm text-destructive">{error}</p> : null}
      {disablingFeature ? (
        <ConfirmDialog
          isOpen
          title={`Turn off ${disablingFeature.label.toLowerCase()}?`}
          description={disablingFeature.disableDescription}
          submitLabel="Turn off"
          onSubmit={() => {
            onToggle(disablingFeature.key, false);
            setDisabling(null);
          }}
          onCancel={() => setDisabling(null)}
          isLoading={pending}
        />
      ) : null}
    </div>
  );
}

function UpgradeSurface({
  canUpgrade,
  upgradePending,
  error,
  onUpgrade,
  onViewBilling,
}: Extract<LabsPreviewProps, { status: 'upgrade' }>) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-4 rounded-lg border border-brand/40 bg-brand/5 p-4">
      <div className="space-y-1">
        <p className="text-sm font-medium">Labs requires a paid plan</p>
        <p className="text-sm text-muted-foreground">
          Upgrade to try durable streams, sandboxes, and serverless.
        </p>
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          className="border-brand text-brand hover:bg-brand/10 hover:text-brand"
          disabled={!canUpgrade || upgradePending}
          onClick={onUpgrade}
        >
          {upgradePending ? <Spinner /> : 'Add a credit card'}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="text-muted-foreground"
          onClick={onViewBilling}
        >
          More details
        </Button>
      </div>
    </div>
  );
}
