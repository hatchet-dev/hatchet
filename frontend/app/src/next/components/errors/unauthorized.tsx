import {
  Alert,
  AlertTitle,
  AlertDescription,
} from '@/next/components/ui/alert';
import { FaLock } from 'react-icons/fa';

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
