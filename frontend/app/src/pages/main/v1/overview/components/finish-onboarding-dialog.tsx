import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';

// Confirming through OK is what hides onboarding and navigates; closing
// the dialog any other way leaves onboarding visible on the Finish tab.
export function FinishOnboardingDialog({
  open,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Onboarding complete</DialogTitle>
        </DialogHeader>
        <p className="text-sm">
          Onboarding has been removed from Overview. You can run it again from
          Settings → General by selecting Restart onboarding.
        </p>
        <DialogFooter>
          <Button
            className="focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ring-offset-background"
            onClick={onConfirm}
          >
            OK
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
