import { SandboxProviderKind } from '@/lib/api/generated/cloud/data-contracts';

export type ConfigField = {
  key: string;
  label: string;
  help: string;
  required?: boolean;
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
  SandboxProviderKind.MODAL,
  SandboxProviderKind.VERCEL,
];

export const KINDS: Record<
  SandboxProviderKind,
  {
    label: string;
    description: string;
    keyLabel?: string;
    keyPlaceholder?: string;
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
  [SandboxProviderKind.MODAL]: {
    label: 'Modal',
    description: 'Sandboxes with filesystem snapshots.',
    keyLabel: 'Token secret',
    keyPlaceholder: 'as-…',
    keyHint: 'The secret half of a Modal API token.',
    templatePlaceholder: 'debian:bookworm-slim',
    fields: [
      {
        key: 'tokenId',
        label: 'Token ID',
        help: 'The ID half of the same Modal API token.',
        required: true,
        type: 'text',
        placeholder: 'ak-…',
        mono: true,
      },
      {
        key: 'app',
        label: 'App',
        help: 'Modal app the sandboxes run under. Created if missing.',
        type: 'text',
        placeholder: 'hatchet-sandboxes',
        mono: true,
      },
      {
        key: 'environment',
        label: 'Environment',
        help: "Modal environment. Leave empty for the token's default.",
        type: 'text',
        mono: true,
      },
      { ...workdir, placeholder: '/workspace' },
      snapshotEveryTurn,
    ],
  },
  [SandboxProviderKind.VERCEL]: {
    label: 'Vercel',
    description: 'MicroVMs with persistent snapshots.',
    keyPlaceholder: 'vcp_…',
    keyHint: 'A Vercel access token.',
    templatePlaceholder: 'vercel/sandbox/universal',
    fields: [
      {
        key: 'teamId',
        label: 'Team ID',
        help: 'The team the access token belongs to.',
        required: true,
        type: 'text',
        placeholder: 'team_…',
        mono: true,
      },
      {
        key: 'projectId',
        label: 'Project',
        help: 'ID or name of the project sandboxes are created in.',
        required: true,
        type: 'text',
        mono: true,
      },
      { ...workdir, placeholder: '/vercel/sandbox' },
      snapshotEveryTurn,
    ],
  },
};

export const SUPPORT_URL =
  'mailto:support@hatchet.run?subject=Enable%20sandboxes';
