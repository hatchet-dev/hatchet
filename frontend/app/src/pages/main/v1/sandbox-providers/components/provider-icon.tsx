import { KINDS } from '../lib/kinds';
import { SandboxProviderKind } from '@/lib/api/generated/cloud/data-contracts';
import { cn } from '@/lib/utils';

const DaytonaIcon = ({ className }: { className?: string }) => (
  <svg
    viewBox="0 0 275 287"
    fill="currentColor"
    className={className}
    aria-hidden="true"
  >
    <path d="M14.5584 193.736H114.275V227.925H14.5584V193.736Z" />
    <path d="M148.464 74.076H262.426V108.265H148.464V74.076Z" />
    <path d="M88.6338 84.6127L173.246 0L197.422 24.175L112.809 108.788L88.6338 84.6127Z" />
    <path d="M89.157 170.084L24.175 105.102L0 129.277L64.9819 194.259L89.157 170.084Z" />
    <path d="M174.629 217.911L106.133 286.407L81.9577 262.232L150.454 193.736L174.629 217.911Z" />
    <path d="M174.106 132.44L250.66 208.994L274.835 184.819L198.281 108.265L174.106 132.44Z" />
    <path d="M88.6338 48.434V131.057H54.4451L54.4451 48.434H88.6338Z" />
    <path d="M208.294 168.094V270.66H174.106V168.094H208.294Z" />
  </svg>
);

const E2BIcon = ({ className }: { className?: string }) => (
  <svg
    viewBox="0 0 24.8 32"
    fill="currentColor"
    className={className}
    aria-hidden="true"
  >
    <path d="M18.6 25.6H6.2V12.8H18.6V25.6Z" opacity="0.4" />
    <path d="M18.6 6.4H6.2V25.6H18.6V6.4ZM24.8 32H0V0H24.8V32Z" />
  </svg>
);

const ArchilIcon = ({ className }: { className?: string }) => (
  <svg viewBox="0 0 64 64" className={className} aria-hidden="true">
    <rect
      x="4"
      y="4"
      width="56"
      height="56"
      fill="none"
      stroke="currentColor"
      strokeWidth="3.5"
    />
    <path
      fill="currentColor"
      transform="translate(6 6) scale(0.83) translate(-115.998 -166.784)"
      d="M172.376 222.438L169.116 222.447V207.734C169.116 206.191 167.865 204.939 166.321 204.938H156.427C155.042 204.939 153.762 205.677 153.067 206.876L144.003 222.525L123.625 222.59L150.84 175.562L172.376 175.41V222.438ZM168.961 184.194C168.961 182.71 167.017 182.155 166.233 183.415L156.512 199.038C155.836 200.124 156.617 201.53 157.896 201.53H166.088C167.675 201.53 168.961 200.244 168.961 198.657V184.194Z"
    />
  </svg>
);

const FreestyleIcon = ({ className }: { className?: string }) => (
  <svg
    viewBox="0 0 347 280"
    fill="none"
    stroke="currentColor"
    strokeWidth="25"
    strokeLinecap="round"
    className={className}
    aria-hidden="true"
  >
    <path d="M70 267V235.793C37.4932 229.296 13 200.594 13 166.177C13 134.93 33.1885 108.399 61.2324 98.9148C61.9277 51.3467 100.705 13 148.438 13C183.979 13 214.554 34.2582 228.143 64.7527C234.182 63.4301 240.454 62.733 246.89 62.733C295.058 62.733 334.105 101.781 334.105 149.949C334.105 182.845 315.893 211.488 289 226.343V267" />
    <path d="M146 237V267" />
    <path d="M215 237V267" />
  </svg>
);

export const ProviderIcon = ({
  kind,
  className,
}: {
  kind: SandboxProviderKind;
  className?: string;
}) => {
  const cls = cn('size-4 shrink-0', className);
  switch (kind) {
    case SandboxProviderKind.DAYTONA:
      return <DaytonaIcon className={cls} />;
    case SandboxProviderKind.E2B:
      return <E2BIcon className={cls} />;
    case SandboxProviderKind.ARCHIL:
      return <ArchilIcon className={cls} />;
    case SandboxProviderKind.FREESTYLE:
      return <FreestyleIcon className={cls} />;
    default:
      const exhaustiveCheck: never = kind;
      throw new Error(`Unhandled sandbox provider kind: ${exhaustiveCheck}`);
  }
};

export const ProviderName = ({
  kind,
  className,
}: {
  kind: SandboxProviderKind;
  className?: string;
}) => (
  <span className={cn('inline-flex items-center gap-x-2', className)}>
    <ProviderIcon kind={kind} />
    {KINDS[kind].label}
  </span>
);
