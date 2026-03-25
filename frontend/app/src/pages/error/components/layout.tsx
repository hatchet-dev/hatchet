import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
} from '@/components/v1/ui/card';
import { cn } from '@/lib/utils';
import { PropsWithChildren, ReactNode } from 'react';

export function ErrorPageLayout({
  title,
  description,
  icon,
  actions,
  children,
  className,
}: PropsWithChildren<{
  title: ReactNode;
  description?: ReactNode;
  icon?: ReactNode;
  actions?: ReactNode;
  className?: string;
}>) {
  return (
    <div className="flex h-full w-full flex-1 flex-row items-center justify-center p-6">
      <Card
        className={cn(
          'w-full max-w-xl border-border/60 bg-background shadow-sm',
          className,
        )}
      >
        <CardHeader className="pb-3 text-center">
          {icon && (
            <div className="mx-auto mb-2 flex h-10 w-10 items-center justify-center rounded-md border bg-muted/30 text-foreground/80">
              {icon}
            </div>
          )}
          <div className="text-xl font-semibold tracking-tight sm:text-2xl">
            {title}
          </div>
          {description && (
            <div className="mx-auto max-w-prose text-sm text-muted-foreground">
              {description}
            </div>
          )}
        </CardHeader>

        {children && (
          <CardContent className="pt-0">
            <div className="space-y-3">{children}</div>
          </CardContent>
        )}

        {actions && (
          <CardFooter className="flex flex-col gap-2 sm:flex-row sm:justify-center">
            {actions}
          </CardFooter>
        )}
      </Card>
    </div>
  );
}
