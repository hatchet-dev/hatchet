import { ReactNode } from 'react';

export function SettingsPageHeader({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children?: ReactNode;
}) {
  return (
    <div className="mb-6 flex flex-col gap-4 border-b border-border/50 pb-6 md:flex-row md:items-start md:justify-between">
      <div className="space-y-1.5">
        <h1 className="text-xl font-semibold tracking-tight">{title}</h1>
        <p className="max-w-xl text-sm text-muted-foreground">{description}</p>
      </div>

      {children}
    </div>
  );
}
