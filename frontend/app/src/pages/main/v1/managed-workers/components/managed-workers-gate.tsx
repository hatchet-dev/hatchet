import { Button } from '@/components/v1/ui/button';
import useControlPlane from '@/hooks/use-control-plane';
import { ErrorPageLayout } from '@/pages/error/components/layout';
import { Cloud, Undo2 } from 'lucide-react';
import { PropsWithChildren } from 'react';

export function ManagedWorkersGate({ children }: PropsWithChildren) {
  const { isControlPlaneEnabled } = useControlPlane();

  if (isControlPlaneEnabled) {
    return <>{children}</>;
  }

  return (
    <ErrorPageLayout
      icon={<Cloud className="h-5 w-5" />}
      title="Managed Workers are not available"
      description="Managed Workers are only available in Hatchet Cloud."
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
