import { formatInvoiceAmount, formatInvoiceDate } from './invoice-formatters';
import { InvoiceLineItems } from './invoice-line-items';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { OrganizationInvoicePreview } from '@/lib/api/generated/control-plane/data-contracts';

interface UpcomingInvoiceDialogProps {
  preview: OrganizationInvoicePreview | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function UpcomingInvoiceDialog({
  preview,
  open,
  onOpenChange,
}: UpcomingInvoiceDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Upcoming invoice</DialogTitle>
          <DialogDescription>
            Estimated charges for the current billing cycle. Totals can change
            as usage is reported.
          </DialogDescription>
        </DialogHeader>

        {preview ? (
          <div className="space-y-4">
            <div className="flex items-baseline justify-between gap-4">
              <p className="text-sm text-muted-foreground">
                Invoice date {formatInvoiceDate(preview.invoiceAt)}
              </p>
              <p className="text-lg font-semibold text-foreground">
                {formatInvoiceAmount(preview.totalCents, preview.currency)}
              </p>
            </div>

            <InvoiceLineItems preview={preview} />

            {preview.subtotalCents !== preview.totalCents ? (
              <div className="flex justify-between text-sm text-muted-foreground">
                <span>Subtotal</span>
                <span>
                  {formatInvoiceAmount(preview.subtotalCents, preview.currency)}
                </span>
              </div>
            ) : null}

            <div className="flex justify-between text-sm font-medium text-foreground">
              <span>Estimated total</span>
              <span>
                {formatInvoiceAmount(preview.totalCents, preview.currency)}
              </span>
            </div>
          </div>
        ) : null}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
