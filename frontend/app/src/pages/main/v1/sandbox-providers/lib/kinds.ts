import { SandboxProviderKind } from '@/lib/api/generated/cloud/data-contracts';

export const KIND_ORDER: SandboxProviderKind[] = [
  SandboxProviderKind.DAYTONA,
  SandboxProviderKind.SUPABASE,
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
    available: boolean;
  }
> = {
  [SandboxProviderKind.DAYTONA]: {
    label: 'Daytona',
    description: 'Daytona sandboxes with snapshots.',
    keyHint:
      'An API key from the Daytona dashboard (Settings → API Keys). Leave the template empty for the default Daytona snapshot.',
    templatePlaceholder: 'daytonaio/sandbox:latest',
    available: true,
  },
  [SandboxProviderKind.SUPABASE]: {
    label: 'Supabase',
    description: 'Supabase sandboxes.',
    keyHint: 'A Supabase API key. The Supabase driver is not available yet.',
    templatePlaceholder: '',
    available: false,
  },
  [SandboxProviderKind.ARCHIL]: {
    label: 'Archil',
    description: 'Archil sandboxes.',
    keyHint: 'An Archil API key. The Archil driver is not available yet.',
    templatePlaceholder: '',
    available: false,
  },
  [SandboxProviderKind.FREESTYLE]: {
    label: 'Freestyle',
    description: 'Freestyle VMs with instant memory and disk snapshots.',
    keyHint:
      'An API key from admin.freestyle.sh. Leave the template empty for the platform default image, or set a snapshot id or slug.',
    templatePlaceholder: 'sh-… or your-snapshot-slug',
    available: true,
  },
};

export const SUPPORT_URL =
  'mailto:support@hatchet.run?subject=Enable%20sandboxes';
