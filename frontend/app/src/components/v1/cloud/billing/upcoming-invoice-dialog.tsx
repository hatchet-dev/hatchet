import {
  formatInvoiceAmount,
  formatInvoiceDate,
  formatInvoicePeriod,
  formatQuantity,
} from './invoice-formatters';
import { Badge } from '@/components/v1/ui/badge';
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

            <div className="divide-y divide-border rounded-md border border-border/50">
              {preview.lineItems.map((item, index) => {
                const period = formatInvoicePeriod(
                  item.periodStart,
                  item.periodEnd,
                );
                return (
                  <div
                    key={`${item.planId}-${item.featureId ?? 'base'}-${index}`}
                    className="p-4 space-y-2"
                  >
                    <div className="flex items-start justify-between gap-4">
                      <div className="min-w-0">
                        <p className="font-medium text-foreground">
                          {item.displayName}
                        </p>
                        <p className="mt-1 text-sm text-muted-foreground">
                          {item.description}
                        </p>
                      </div>
                      <p className="shrink-0 text-sm font-medium text-foreground">
                        {formatInvoiceAmount(item.totalCents, preview.currency)}
                      </p>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant="queued">
                        Qty: {formatQuantity(item.quantity)}
                      </Badge>
                      {period ? (
                        <span className="text-xs text-muted-foreground">
                          {period}
                        </span>
                      ) : null}
                    </div>
                  </div>
                );
              })}
            </div>

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
