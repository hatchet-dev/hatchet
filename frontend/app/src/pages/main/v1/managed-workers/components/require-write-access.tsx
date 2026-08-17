import { Button } from '@/components/v1/ui/button';
import useCanWrite from '@/hooks/use-can-write';
import { ErrorPageLayout } from '@/pages/error/components/layout';
import { Lock, Undo2 } from 'lucide-react';
import { PropsWithChildren } from 'react';

export function RequireManagedComputeWriteAccess({
  children,
}: PropsWithChildren) {
  const canWrite = useCanWrite();

  if (canWrite) {
    return <>{children}</>;
  }

  return (
    <ErrorPageLayout
      icon={<Lock className="h-5 w-5" />}
      title="You don't have access to set up Managed Compute"
      description="You must be an owner, admin, or member of this tenant to set up Managed Compute. Contact a tenant admin if you need access."
      actions={
        <Button
          leftIcon={<Undo2 className="h-4 w-4" />}
          variant="outline"
          onClick={() => window.history.back()}
        >
          Go back
        </Button>
      }
    />
  );
}
