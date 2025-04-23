import { ManagedWorkerRegion } from '@/lib/api/generated/cloud/data-contracts';

export interface MachineType {
  title: string;
  cpuKind: string;
  cpus: number;
  memoryMb: number;
}

export const DefaultMachineTypes: MachineType[] = [
  {
    title: '1 CPU, 1 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 1,
    memoryMb: 1024,
  },
  {
    title: '1 CPU, 2 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 1,
    memoryMb: 2048,
  },
  {
    title: '2 CPU, 2 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 2,
    memoryMb: 2048,
  },
  {
    title: '2 CPU, 4 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 2,
    memoryMb: 4096,
  },
  {
    title: '4 CPU, 8 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 4,
    memoryMb: 8192,
  },
  {
    title: '8 CPU, 16 GB RAM (shared CPU)',
    cpuKind: 'shared',
    cpus: 8,
    memoryMb: 16384,
  },
  {
    title: '1 CPU, 1 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 1,
    memoryMb: 1024,
  },
  {
    title: '1 CPU, 2 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 1,
    memoryMb: 2048,
  },
  {
    title: '2 CPU, 2 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 2,
    memoryMb: 2048,
  },
  {
    title: '2 CPU, 4 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 2,
    memoryMb: 4096,
  },
  {
    title: '4 CPU, 8 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 4,
    memoryMb: 8192,
  },
  {
    title: '8 CPU, 16 GB RAM (performance CPU)',
    cpuKind: 'performance',
    cpus: 8,
    memoryMb: 16384,
  },
];

export type Region = {
  name: string;
  value: ManagedWorkerRegion;
  emoji: string;
};

export const regions: Region[] = [
  {
    name: 'Amsterdam, Netherlands',
    value: ManagedWorkerRegion.Ams,
    emoji: '🇳🇱',
  },
  {
    name: 'Stockholm, Sweden',
    value: ManagedWorkerRegion.Arn,
    emoji: '🇸🇪',
  },
  {
    name: 'Atlanta, Georgia (US)',
    value: ManagedWorkerRegion.Atl,
    emoji: '🇺🇸',
  },
  {
    name: 'Bogotá, Colombia',
    value: ManagedWorkerRegion.Bog,
    emoji: '🇨🇴',
  },
  {
    name: 'Boston, Massachusetts (US)',
    value: ManagedWorkerRegion.Bos,
    emoji: '🇺🇸',
  },
  {
    name: 'Paris, France',
    value: ManagedWorkerRegion.Cdg,
    emoji: '🇫🇷',
  },
  {
    name: 'Denver, Colorado (US)',
    value: ManagedWorkerRegion.Den,
    emoji: '🇺🇸',
  },
  {
    name: 'Dallas, Texas (US)',
    value: ManagedWorkerRegion.Dfw,
    emoji: '🇺🇸',
  },
  {
    name: 'Secaucus, NJ (US)',
    value: ManagedWorkerRegion.Ewr,
    emoji: '🇺🇸',
  },
  {
    name: 'Ezeiza, Argentina',
    value: ManagedWorkerRegion.Eze,
    emoji: '🇦🇷',
  },
  {
    name: 'Guadalajara, Mexico',
    value: ManagedWorkerRegion.Gdl,
    emoji: '🇲🇽',
  },
  {
    name: 'Rio de Janeiro, Brazil',
    value: ManagedWorkerRegion.Gig,
    emoji: '🇧🇷',
  },
  {
    name: 'Sao Paulo, Brazil',
    value: ManagedWorkerRegion.Gru,
    emoji: '🇧🇷',
  },
  {
    name: 'Hong Kong, Hong Kong',
    value: ManagedWorkerRegion.Hkg,
    emoji: '🇭🇰',
  },
  {
    name: 'Ashburn, Virginia (US)',
    value: ManagedWorkerRegion.Iad,
    emoji: '🇺🇸',
  },
  {
    name: 'Johannesburg, South Africa',
    value: ManagedWorkerRegion.Jnb,
    emoji: '🇿🇦',
  },
  {
    name: 'Los Angeles, California (US)',
    value: ManagedWorkerRegion.Lax,
    emoji: '🇺🇸',
  },
  {
    name: 'London, United Kingdom',
    value: ManagedWorkerRegion.Lhr,
    emoji: '🇬🇧',
  },
  {
    name: 'Madrid, Spain',
    value: ManagedWorkerRegion.Mad,
    emoji: '🇪🇸',
  },
  {
    name: 'Miami, Florida (US)',
    value: ManagedWorkerRegion.Mia,
    emoji: '🇺🇸',
  },
  {
    name: 'Tokyo, Japan',
    value: ManagedWorkerRegion.Nrt,
    emoji: '🇯🇵',
  },
  {
    name: 'Chicago, Illinois (US)',
    value: ManagedWorkerRegion.Ord,
    emoji: '🇺🇸',
  },
  {
    name: 'Bucharest, Romania',
    value: ManagedWorkerRegion.Otp,
    emoji: '🇷🇴',
  },
  {
    name: 'Phoenix, Arizona (US)',
    value: ManagedWorkerRegion.Phx,
    emoji: '🇺🇸',
  },
  {
    name: 'Querétaro, Mexico',
    value: ManagedWorkerRegion.Qro,
    emoji: '🇲🇽',
  },
  {
    name: 'Santiago, Chile',
    value: ManagedWorkerRegion.Scl,
    emoji: '🇨🇱',
  },
  {
    name: 'Seattle, Washington (US)',
    value: ManagedWorkerRegion.Sea,
    emoji: '🇺🇸',
  },
  {
    name: 'Singapore, Singapore',
    value: ManagedWorkerRegion.Sin,
    emoji: '🇸🇬',
  },
  {
    name: 'San Jose, California (US)',
    value: ManagedWorkerRegion.Sjc,
    emoji: '🇺🇸',
  },
  {
    name: 'Sydney, Australia',
    value: ManagedWorkerRegion.Syd,
    emoji: '🇦🇺',
  },
  {
    name: 'Warsaw, Poland',
    value: ManagedWorkerRegion.Waw,
    emoji: '🇵🇱',
  },
  {
    name: 'Montreal, Canada',
    value: ManagedWorkerRegion.Yul,
    emoji: '🇨🇦',
  },
  {
    name: 'Toronto, Canada',
    value: ManagedWorkerRegion.Yyz,
    emoji: '🇨🇦',
  },
];

export type ScalingType = 'Autoscaling' | 'Static';
export const scalingTypes: ScalingType[] = ['Static', 'Autoscaling'];
