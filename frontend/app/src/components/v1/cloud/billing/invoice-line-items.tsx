import {
  formatInvoiceAmount,
  formatInvoicePeriod,
  formatQuantity,
} from './invoice-formatters';
import { Badge } from '@/components/v1/ui/badge';
import { OrganizationInvoicePreview } from '@/lib/api/generated/control-plane/data-contracts';

export function InvoiceLineItems({
  preview,
}: {
  preview: OrganizationInvoicePreview;
}) {
  return (
    <div className="divide-y divide-border rounded-md border border-border/50">
      {preview.lineItems.map((item, index) => {
        const period = formatInvoicePeriod(item.periodStart, item.periodEnd);
        return (
          <div
            key={`${item.planId}-${item.featureId ?? 'base'}-${index}`}
            className="space-y-2 p-4"
          >
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0">
                <p className="font-medium text-foreground">
                  {item.displayName}
                </p>
                {item.description ? (
                  <p className="mt-1 text-sm text-muted-foreground">
                    {item.description}
                  </p>
                ) : null}
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
                <span className="text-xs text-muted-foreground">{period}</span>
              ) : null}
            </div>
          </div>
        );
      })}
    </div>
  );
}
