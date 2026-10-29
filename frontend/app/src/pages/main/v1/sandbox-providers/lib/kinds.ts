import { SandboxProviderKind } from '@/lib/api/generated/cloud/data-contracts';

export const KIND_ORDER: SandboxProviderKind[] = [
  SandboxProviderKind.E2B,
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
  [SandboxProviderKind.E2B]: {
    label: 'E2B',
    description: 'Firecracker microVMs with snapshots.',
    keyHint:
      'An API key from the E2B dashboard (Team → API Keys). The default template is "base".',
    templatePlaceholder: 'base',
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
