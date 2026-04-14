import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { useSsoErrorAlert } from '@/hooks/sso/SsoSetupHooks';

export function SsoErrorAlert() {
  const { message } = useSsoErrorAlert();

  if (!message) {
    return null;
  }

  return (
    <Alert variant="destructive">
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}
