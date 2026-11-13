import { SandboxProviderKind } from '@/lib/api/generated/cloud/data-contracts';

export const KIND_ORDER: SandboxProviderKind[] = [
  SandboxProviderKind.DAYTONA,
  SandboxProviderKind.E2B,
  SandboxProviderKind.ARCHIL,
  SandboxProviderKind.FREESTYLE,
];

export const KINDS: Record<
  SandboxProviderKind,
  {
    label: string;
    description: string;
    keyHint: string;
    templatePlaceholder: string;
    configKeys: string[];
  }
> = {
  [SandboxProviderKind.DAYTONA]: {
    label: 'Daytona',
    description: 'Daytona sandboxes with snapshots.',
    keyHint:
      'An API key from the Daytona dashboard (Settings → API Keys). Leave the template empty for the default Daytona snapshot.',
    templatePlaceholder: 'daytonaio/sandbox:latest',
    configKeys: ['template', 'workdir', 'snapshotEveryTurn'],
  },
  [SandboxProviderKind.E2B]: {
    label: 'E2B',
    description: 'E2B sandboxes with filesystem snapshots.',
    keyHint:
      'An API key from the E2B dashboard (Team → API Keys). Leave the template empty for the E2B base template, or set a template name or snapshot id. Sessions are capped at one hour unless the provider config sets "maxTimeoutSeconds" for a plan that allows more.',
    templatePlaceholder: 'base',
    configKeys: [
      'apiUrl',
      'template',
      'workdir',
      'maxTimeoutSeconds',
      'snapshotEveryTurn',
    ],
  },
  [SandboxProviderKind.ARCHIL]: {
    label: 'Archil',
    description: 'Archil persistent microVMs with forks.',
    keyHint:
      'An API key from the Archil console. Keys are scoped to one region; set "region" in the provider config (default aws-us-east-1). The template is an OCI image.',
    templatePlaceholder: 'node:24-bookworm',
    configKeys: ['region', 'template', 'workdir', 'snapshotEveryTurn'],
  },
  [SandboxProviderKind.FREESTYLE]: {
    label: 'Freestyle',
    description: 'Freestyle VMs with instant memory and disk snapshots.',
    keyHint:
      'An API key from admin.freestyle.sh. Leave the template empty for the platform default image, or set a snapshot id or slug.',
    templatePlaceholder: 'sh-… or your-snapshot-slug',
    configKeys: ['template', 'workdir', 'snapshotEveryTurn'],
  },
};

export const SUPPORT_URL =
  'mailto:support@hatchet.run?subject=Enable%20sandboxes';
