import { PropsWithChildren } from 'react';
import useUser from '@/next/hooks/use-user';
import { Button } from '@/components/ui/button';
import { ROUTES } from '@/next/lib/routes';
import { Link } from 'react-router-dom';
import { cn } from '@/lib/utils';
import { Card } from '@/components/ui/card';

interface SignInRequiredActionProps extends PropsWithChildren {
  reject?: React.ReactNode;
  actions?: React.ReactNode;
  variant?: 'default' | 'card';
  className?: string;
  title?: string;
  description?: string;
}

function DefaultActions() {
  return (
    <div className="flex flex-row gap-3">
      <Button variant="outline" size="lg" className="flex-1" asChild>
        <Link to={ROUTES.auth.register}>Sign up</Link>
      </Button>
      <Button variant="default" size="lg" className="flex-1" asChild>
        <Link to={ROUTES.auth.login}>Sign in</Link>
      </Button>
    </div>
  );
}

export function SignInRequiredAction({
  children,
  actions = <DefaultActions />,
  variant = 'default',
  className,
  title = 'Sign into Hatchet Cloud to follow along.',
  description,
}: SignInRequiredActionProps) {
  const { data: user } = useUser();

  if (!user) {
    const content = (
      <>
        <div>{actions}</div>
      </>
    );

    if (variant === 'card') {
      return (
        <Card className={cn('p-8 bg-card/50 border-muted', className)}>
          <h2 className="text-lg font-semibold mb-6">{title}</h2>
          {description && <p className="mb-6">{description}</p>}
          {content}
        </Card>
      );
    }

    return <div className={className}>{content}</div>;
  }

  return <>{children}</>;
}
