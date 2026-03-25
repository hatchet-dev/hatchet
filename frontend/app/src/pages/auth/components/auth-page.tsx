import useApiMeta from '../hooks/use-api-meta';
import useErrorParam from '../hooks/use-error-param';
import { AuthLayout } from './auth-layout';
import { AuthLegalText } from './auth-legal-text';
import { OrContinueWith, SocialAuthButtons } from './social-auth';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { Loading } from '@/components/v1/ui/loading';
import React from 'react';

export function AuthPage({
  title,
  basicSection,
  altAction,
}: {
  title: string;
  basicSection: React.ReactNode;
  altAction: React.ReactNode;
}) {
  useErrorParam();
  const { meta, isLoading } = useApiMeta();

  if (isLoading) {
    return <Loading />;
  }

  const schemes = meta?.auth?.schemes || [];
  const basicEnabled = schemes.includes('basic');
  const googleEnabled = schemes.includes('google');
  const githubEnabled = schemes.includes('github');

  const providers = [
    googleEnabled && 'google',
    githubEnabled && 'github',
  ].filter(Boolean) as Array<'google' | 'github'>;

  const sections = [
    providers.length > 0 && <SocialAuthButtons providers={providers} />,
    basicEnabled && basicSection,
  ].filter(Boolean);

  return (
    <AuthLayout>
      <div className="flex flex-col gap-3 text-center lg:text-left w-full">
        <div className="flex justify-center pb-3 lg:hidden">
          <HatchetLogo className="h-8 w-auto" />
        </div>
        <div className="flex w-full flex-col items-center gap-2 lg:flex-row lg:items-center lg:justify-between">
          <h2
            className="text-2xl font-semibold tracking-tight"
            data-cy="auth-title"
          >
            {title}
          </h2>
          <div className="text-sm text-muted-foreground text-center lg:text-right">
            {altAction}
          </div>
        </div>
      </div>

      {sections.map((section, index) => (
        <React.Fragment key={index}>
          {section}
          {index < sections.length - 1 && <OrContinueWith />}
        </React.Fragment>
      ))}

      <div className="pt-3 space-y-5">
        <AuthLegalText />
      </div>
    </AuthLayout>
  );
}
