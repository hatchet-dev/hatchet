import { Button } from '@/components/v1/ui/button';
import { Icons } from '@/components/v1/ui/icons';
import { ArrowLeft, LockOpen } from 'lucide-react';
import React, { useState } from 'react';

export type SocialAuthProvider = 'google' | 'github' | 'sso';

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
  sso: {
    href: '/api/v1/users/sso/start',
    label: 'SSO',
    icon: <LockOpen className="size-4" />,
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
  ssoExpanded,
  setSsoExpanded,
}: {
  provider: SocialAuthProvider;
  ssoExpanded: boolean;
  setSsoExpanded: any;
}) {
  const cfg = PROVIDER_CONFIG[provider];
  const [email, setEmail] = useState('');

  if (provider === 'sso') {
    return (
      <div className="w-full flex flex-col gap-2">
        {!ssoExpanded && (
          <Button
            variant="outline"
            type="button"
            fullWidth
            onClick={() => setSsoExpanded(true)}
            className="h-11 justify-center gap-2 border-muted-foreground/20 bg-background shadow-sm hover:bg-muted/40"
          >
            {cfg.icon}
            {cfg.label}
          </Button>
        )}
        {ssoExpanded && (
          <div className="flex gap-2">
            <input
              type="text"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="Enter your email"
              className="h-10 flex-1 rounded-md border border-muted-foreground/20 bg-background px-3 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
        )}
        {ssoExpanded && (
          <a
            href={
              email ? `${cfg.href}?email=${encodeURIComponent(email)}` : '#'
            }
            onClick={(e) => {
              if (!email) {
                e.preventDefault();
              }
            }}
            tabIndex={email ? 0 : -1}
          >
            <Button
              variant="outline"
              type="button"
              fullWidth
              disabled={!email}
              className="h-10 border-muted-foreground/20 bg-background shadow-sm hover:bg-muted/40"
            >
              Continue
            </Button>
          </a>
        )}
        {ssoExpanded && (
          <Button
            variant="ghost"
            type="button"
            size="sm"
            onClick={() => setSsoExpanded(false)}
            className="h-11 justify-left gap-1 border-muted-foreground/20 bg-background shadow-sm hover:bg-muted/40"
          >
            <ArrowLeft className="size-4" />
            Back
          </Button>
        )}
      </div>
    );
  }

  return (
    !ssoExpanded && (
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
    )
  );
}

export function SocialAuthButtons({
  providers,
  ssoExpanded,
  setSsoExpanded,
}: {
  providers: SocialAuthProvider[];
  ssoExpanded: boolean;
  setSsoExpanded: any;
}) {
  if (providers.length === 0) {
    return null;
  }
  return (
    <div className="grid sm:grid-flow-col gap-3">
      {providers.map((p) => (
        <SocialAuthButton
          key={p}
          provider={p}
          ssoExpanded={ssoExpanded}
          setSsoExpanded={setSsoExpanded}
        />
      ))}
    </div>
  );
}
