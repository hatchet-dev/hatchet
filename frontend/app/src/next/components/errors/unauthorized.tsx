import {
  Alert,
  AlertTitle,
  AlertDescription,
} from '@/next/components/ui/alert';
import useTenant from '@/next/hooks/use-tenant';
import { FaLock } from 'react-icons/fa';
import { Button } from '../ui/button';

export function Unauthorized() {
  return (
    <div className="max-w-md mx-auto pt-20">
      <Alert>
        <FaLock className="h-4 w-4" />
        <AlertTitle className="text-lg">Unauthorized</AlertTitle>
        <AlertDescription>
          You don't have access to this tenant. Please select a different tenant
          or contact your administrator.
        </AlertDescription>
      </Alert>
    </div>
  );
}

export function WrongTenant({ desiredTenantId }: { desiredTenantId: string }) {
  const { setTenant } = useTenant();

  return (
    <div className="max-w-md mx-auto pt-20">
      <Alert>
        <FaLock className="h-4 w-4" />
        <AlertTitle className="text-lg">Wrong Tenant</AlertTitle>
        <AlertDescription>
          You are trying to access a run from a different tenant context, please
          select the correct tenant to continue.
          <div className="flex flex-row gap-2 mt-2">
            <Button
              onClick={() => {
                setTenant(desiredTenantId);
              }}
            >
              Switch Tenant
            </Button>
          </div>
        </AlertDescription>
      </Alert>
    </div>
  );
}
