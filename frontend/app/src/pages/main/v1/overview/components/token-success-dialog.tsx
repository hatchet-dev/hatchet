import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { SecretCopier } from '@/components/v1/ui/secret-copier';

export function TokenSuccessDialog({
  open,
  onOpenChange,
  token,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  token?: string;
}) {
  if (!open || !token) {
    return null;
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-fit max-w-[700px]">
        <DialogHeader>
          <DialogTitle>Keep it secret, keep it safe</DialogTitle>
        </DialogHeader>
        <p className="text-sm">
          This is the only time we will show you this token. Make sure to copy
          it somewhere safe.
        </p>
        <SecretCopier
          secrets={{ HATCHET_CLIENT_TOKEN: token }}
          className="text-sm"
          maxWidth={'calc(700px - 4rem)'}
          copy
        />
      </DialogContent>
    </Dialog>
  );
}
