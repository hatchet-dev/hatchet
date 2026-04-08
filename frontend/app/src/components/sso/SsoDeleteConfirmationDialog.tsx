import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
} from "@/components/ui/dialog";
import { useSsoDeleteConfirmation } from "@/hooks/sso/SsoSetupHooks";

export function SsoDeleteConfirmationDialog() {
    const { open, onOpenChange, onConfirm, deleting, providerName } = useSsoDeleteConfirmation();
    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Delete SSO Configuration?</DialogTitle>
                    <DialogDescription>
                        This will permanently delete the SSO configuration for <strong>{providerName}</strong>. This
                        action cannot be undone.
                    </DialogDescription>
                </DialogHeader>
                <DialogFooter>
                    <Button variant="outline" onClick={() => onOpenChange(false)} disabled={deleting}>
                        Cancel
                    </Button>
                    <Button variant="destructive" onClick={onConfirm} disabled={deleting}>
                        {deleting ? (
                            <>
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                Deleting...
                            </>
                        ) : (
                            "Delete Configuration"
                        )}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
