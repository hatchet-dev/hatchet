import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';

// Confirming is what hides onboarding; Cancel, Escape, or clicking outside
// leaves it visible and persists nothing. Unlike the Finish dialog, a
// confirmed skip captures no completion event and stays on Overview.
export function SkipOnboardingDialog({
  open,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  const focusRing =
    'focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ring-offset-background';

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Skip onboarding?</DialogTitle>
        </DialogHeader>
        <p className="text-sm">
          Onboarding will be removed from Overview. You can run it again from
          Settings → General by selecting Restart onboarding.
        </p>
        <DialogFooter>
          <Button
            variant="ghost"
            className={focusRing}
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button className={focusRing} onClick={onConfirm}>
            Skip onboarding
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
