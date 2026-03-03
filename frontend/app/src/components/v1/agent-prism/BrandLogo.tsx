import { type FC } from "react";

const OpenAILogo: FC<{ className?: string }> = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="currentColor">
    <path d="M22.282 9.821a5.985 5.985 0 0 0-.516-4.91 6.046 6.046 0 0 0-6.51-2.9A6.065 6.065 0 0 0 4.981 4.18a5.985 5.985 0 0 0-3.998 2.9 6.046 6.046 0 0 0 .743 7.097 5.98 5.98 0 0 0 .51 4.911 6.051 6.051 0 0 0 6.515 2.9A5.985 5.985 0 0 0 13.26 24a6.056 6.056 0 0 0 5.772-4.206 5.99 5.99 0 0 0 3.997-2.9 6.056 6.056 0 0 0-.747-7.073zM13.26 22.43a4.476 4.476 0 0 1-2.876-1.04l.141-.081 4.779-2.758a.795.795 0 0 0 .392-.681v-6.737l2.02 1.168a.071.071 0 0 1 .038.052v5.583a4.504 4.504 0 0 1-4.494 4.494zM3.6 18.304a4.47 4.47 0 0 1-.535-3.014l.142.085 4.783 2.759a.771.771 0 0 0 .78 0l5.843-3.369v2.332a.08.08 0 0 1-.033.062L9.74 19.95a4.5 4.5 0 0 1-6.14-1.646zM2.34 7.896a4.485 4.485 0 0 1 2.366-1.973V11.6a.766.766 0 0 0 .388.676l5.815 3.355-2.02 1.168a.076.076 0 0 1-.071 0l-4.83-2.786A4.504 4.504 0 0 1 2.34 7.872zm16.597 3.855l-5.833-3.387L15.119 7.2a.076.076 0 0 1 .071 0l4.83 2.791a4.494 4.494 0 0 1-.676 8.105v-5.678a.79.79 0 0 0-.407-.667zm2.01-3.023l-.141-.085-4.774-2.782a.776.776 0 0 0-.785 0L9.409 9.23V6.897a.066.066 0 0 1 .028-.061l4.83-2.787a4.5 4.5 0 0 1 6.68 4.66zm-12.64 4.135l-2.02-1.164a.08.08 0 0 1-.038-.057V6.075a4.5 4.5 0 0 1 7.375-3.453l-.142.08L8.704 5.46a.795.795 0 0 0-.393.681zm1.097-2.365l2.602-1.5 2.607 1.5v2.999l-2.597 1.5-2.607-1.5z" />
  </svg>
);

const AnthropicLogo: FC<{ className?: string }> = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="currentColor">
    <path d="M15.5 2.694h5.97l-9.204 18.612h-5.97L15.5 2.694zm-7.112 0h5.515l-9.177 18.612H0L8.388 2.694z" />
  </svg>
);

const GoogleLogo: FC<{ className?: string }> = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="none">
    <path
      d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
      fill="#4285F4"
    />
    <path
      d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
      fill="#34A853"
    />
    <path
      d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
      fill="#FBBC05"
    />
    <path
      d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
      fill="#EA4335"
    />
  </svg>
);

const MetaLogo: FC<{ className?: string }> = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="currentColor">
    <path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z" />
  </svg>
);

const MistralLogo: FC<{ className?: string }> = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="currentColor">
    <rect width="4" height="4" x="0" y="0" />
    <rect width="4" height="4" x="5" y="0" />
    <rect width="4" height="4" x="10" y="0" />
    <rect width="4" height="4" x="15" y="0" />
    <rect width="4" height="4" x="20" y="0" />
    <rect width="4" height="4" x="0" y="5" />
    <rect width="4" height="4" x="15" y="5" />
    <rect width="4" height="4" x="20" y="5" />
    <rect width="4" height="4" x="0" y="10" />
    <rect width="4" height="4" x="10" y="10" />
    <rect width="4" height="4" x="15" y="10" />
    <rect width="4" height="4" x="20" y="10" />
    <rect width="4" height="4" x="0" y="15" />
    <rect width="4" height="4" x="5" y="15" />
    <rect width="4" height="4" x="15" y="15" />
    <rect width="4" height="4" x="20" y="15" />
    <rect width="4" height="4" x="0" y="20" />
    <rect width="4" height="4" x="5" y="20" />
    <rect width="4" height="4" x="10" y="20" />
    <rect width="4" height="4" x="15" y="20" />
    <rect width="4" height="4" x="20" y="20" />
  </svg>
);

const PerplexityLogo: FC<{ className?: string }> = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="currentColor">
    <path d="M13.913.5v10.203L23.413 5.5zm-3.826 0L.587 5.5l9.5 5.203zm0 23L.587 18.5l9.5-5.203zm3.826 0v-10.203L23.413 18.5z" />
  </svg>
);

// Logo registry
const LOGO_REGISTRY = {
  openai: OpenAILogo,
  anthropic: AnthropicLogo,
  google: GoogleLogo,
  meta: MetaLogo,
  mistral: MistralLogo,
  perplexity: PerplexityLogo,
} as const;

type BrandType = keyof typeof LOGO_REGISTRY;

type BrandLogoProps = {
  brand: BrandType | string;
  className?: string;
  fallback?: React.ReactNode;
};

export const BrandLogo: FC<BrandLogoProps> = ({
  brand,
  className = "size-4",
  fallback = null,
}) => {
  const Logo = LOGO_REGISTRY[brand as BrandType];

  if (!Logo) return <>{fallback}</>;

  return <Logo className={className} />;
};
