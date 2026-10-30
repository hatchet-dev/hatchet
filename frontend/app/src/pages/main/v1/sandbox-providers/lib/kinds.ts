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
    description: 'Freestyle VMs.',
    keyHint: 'A Freestyle API key. The Freestyle driver is not available yet.',
    templatePlaceholder: '',
    available: false,
  },
};

export const SUPPORT_URL =
  'mailto:support@hatchet.run?subject=Enable%20sandboxes';
