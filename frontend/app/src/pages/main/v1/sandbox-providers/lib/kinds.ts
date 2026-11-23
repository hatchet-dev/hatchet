import { SandboxProviderKind } from '@/lib/api/generated/cloud/data-contracts';

export type ConfigField = {
  key: string;
  label: string;
  help: string;
} & (
  | { type: 'text'; placeholder?: string; mono?: boolean }
  | { type: 'number'; placeholder?: string }
  | { type: 'boolean' }
  | { type: 'select'; options: string[] }
);

const workdir: ConfigField = {
  key: 'workdir',
  label: 'Working directory',
  help: 'Where commands run. Must be an absolute path.',
  type: 'text',
  placeholder: '/home/user',
  mono: true,
};

const snapshotEveryTurn: ConfigField = {
  key: 'snapshotEveryTurn',
  label: 'Snapshot every turn',
  help: 'Slower, but every command can be replayed.',
  type: 'boolean',
};

export const ARCHIL_REGIONS = [
  'aws-us-east-1',
  'aws-us-west-2',
  'aws-eu-west-1',
  'gcp-us-central1',
];

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
    fields: ConfigField[];
  }
> = {
  [SandboxProviderKind.DAYTONA]: {
    label: 'Daytona',
    description: 'Sandboxes with snapshots.',
    keyHint: 'From the Daytona dashboard under Settings and API Keys.',
    templatePlaceholder: 'daytonaio/sandbox:latest',
    fields: [workdir, snapshotEveryTurn],
  },
  [SandboxProviderKind.E2B]: {
    label: 'E2B',
    description: 'Sandboxes with filesystem snapshots.',
    keyHint:
      'From the E2B dashboard under Team and API Keys. On the Hobby plan keep max idle at or below 3600, E2B rejects longer.',
    templatePlaceholder: 'base',
    fields: [
      {
        key: 'apiUrl',
        label: 'API URL',
        help: 'Only for self hosted E2B.',
        type: 'text',
        placeholder: 'https://api.e2b.dev',
        mono: true,
      },
      workdir,
      snapshotEveryTurn,
    ],
  },
  [SandboxProviderKind.ARCHIL]: {
    label: 'Archil',
    description: 'Persistent microVMs with forks.',
    keyHint: 'From the Archil console. A key works in one region only.',
    templatePlaceholder: 'node:24-bookworm',
    fields: [
      {
        key: 'region',
        label: 'Region',
        help: 'Must match the region of the API key.',
        type: 'select',
        options: ARCHIL_REGIONS,
      },
      workdir,
      snapshotEveryTurn,
    ],
  },
  [SandboxProviderKind.FREESTYLE]: {
    label: 'Freestyle',
    description: 'VMs with instant memory and disk snapshots.',
    keyHint: 'From admin.freestyle.sh.',
    templatePlaceholder: 'sh-… or a snapshot slug',
    fields: [workdir, snapshotEveryTurn],
  },
};

export const SUPPORT_URL =
  'mailto:support@hatchet.run?subject=Enable%20sandboxes';
