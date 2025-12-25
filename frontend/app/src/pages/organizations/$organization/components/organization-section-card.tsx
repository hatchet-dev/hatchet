import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/v1/ui/card';
import { ReactNode } from 'react';

type OrganizationSectionCardProps = {
  title: string;
  description?: string;
  action?: ReactNode;
  children: ReactNode;
};

export function OrganizationSectionCard({
  title,
  description,
  action,
  children,
}: OrganizationSectionCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between gap-4">
          <span className="min-w-0 truncate">{title}</span>
          {action ? <div className="shrink-0">{action}</div> : null}
        </CardTitle>
        {description ? <CardDescription>{description}</CardDescription> : null}
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  );
}


