import useApiMeta from '../hooks/use-api-meta';
import useErrorParam from '../hooks/use-error-param';
import { AuthLayout } from './auth-layout';
import { AuthLegalText } from './auth-legal-text';
import { OrContinueWith, SocialAuthButtons } from './social-auth';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { Loading } from '@/components/v1/ui/loading';
import React from 'react';

type AuthPagePromptFn = (opts: {
  basicEnabled: boolean;
  googleEnabled: boolean;
  githubEnabled: boolean;
}) => string;

export function AuthPage({
  title,
  promptFn,
  basicSection,
  footer,
}: {
  title: string;
  promptFn: AuthPagePromptFn;
  basicSection: React.ReactNode;
  footer: React.ReactNode;
}) {
  useErrorParam();
  const meta = useApiMeta();

  if (meta.isLoading) {
    return <Loading />;
  }

  const schemes = meta.data?.auth?.schemes || [];
  const basicEnabled = schemes.includes('basic');
  const googleEnabled = schemes.includes('google');
  const githubEnabled = schemes.includes('github');

  const providers = [
    googleEnabled && 'google',
    githubEnabled && 'github',
  ].filter(Boolean) as Array<'google' | 'github'>;

  const prompt = promptFn({ basicEnabled, googleEnabled, githubEnabled });

  const sections = [
    providers.length > 0 && <SocialAuthButtons providers={providers} />,
    basicEnabled && basicSection,
  ].filter(Boolean);

  return (
    <AuthLayout>
      <div className="flex flex-col space-y-4 text-center lg:text-left max-w-md">
        <div className="flex justify-center lg:justify-start pb-4">
          <HatchetLogo className="h-6 w-auto" />
        </div>
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        <p className="text-sm text-gray-700 dark:text-gray-300">{prompt}</p>
        {footer}
      </div>

      {sections.map((section, index) => (
        <React.Fragment key={index}>
          {section}
          {index < sections.length - 1 && <OrContinueWith />}
        </React.Fragment>
      ))}

      <div className="pt-4 space-y-5">
        <AuthLegalText />
      </div>
    </AuthLayout>
  );
}
