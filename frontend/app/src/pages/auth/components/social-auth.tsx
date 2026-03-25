import { Button } from '@/components/v1/ui/button';
import { Icons } from '@/components/v1/ui/icons';
import React from 'react';

export type SocialAuthProvider = 'google' | 'github';

const PROVIDER_CONFIG: Record<
  SocialAuthProvider,
  { href: string; label: string; icon: React.ReactNode }
> = {
  google: {
    href: '/api/v1/users/google/start',
    label: 'Google',
    icon: <Icons.google className="size-4" />,
  },
  github: {
    href: '/api/v1/users/github/start',
    label: 'GitHub',
    icon: <Icons.gitHub className="size-4" />,
  },
};

export function OrContinueWith() {
  return (
    <div className="relative my-4">
      <div className="absolute inset-0 flex items-center">
        <span className="w-full border-t border-muted-foreground/20" />
      </div>
      <div className="relative flex justify-center text-xs uppercase">
        <span className="bg-background px-2 text-muted-foreground">
          Or continue with
        </span>
      </div>
    </div>
  );
}

export function SocialAuthButton({
  provider,
}: {
  provider: SocialAuthProvider;
}) {
  const cfg = PROVIDER_CONFIG[provider];

  return (
    <a href={cfg.href} className="w-full">
      <Button
        variant="outline"
        type="button"
        fullWidth
        className="h-11 justify-center gap-2 border-muted-foreground/20 bg-background shadow-sm hover:bg-muted/40"
      >
        {cfg.icon}
        {cfg.label}
      </Button>
    </a>
  );
}

export function SocialAuthButtons({
  providers,
}: {
  providers: SocialAuthProvider[];
}) {
  if (providers.length === 0) {
    return null;
  }

  return (
    <div className="grid sm:grid-flow-col gap-3">
      {providers.map((p) => (
        <SocialAuthButton key={p} provider={p} />
      ))}
    </div>
  );
}
