import {
  formatInvoiceAmount,
  formatInvoiceDate,
  formatPlanIds,
  invoiceStatusBadge,
} from './invoice-formatters';
import { UpcomingInvoiceDialog } from './upcoming-invoice-dialog';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Skeleton } from '@/components/v1/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import {
  OrganizationInvoice,
  OrganizationInvoicePreview,
} from '@/lib/api/generated/control-plane/data-contracts';
import { ArrowTopRightOnSquareIcon } from '@heroicons/react/24/outline';
import { useMemo, useState } from 'react';

interface InvoicesProps {
  invoicePreviews?: OrganizationInvoicePreview[];
  invoices?: OrganizationInvoice[];
  isLoading?: boolean;
  isError?: boolean;
  onRetry?: () => void;
}

function InvoiceSectionLabel({ children }: { children: string }) {
  return (
    <span className="inline-flex items-center rounded-full bg-muted px-2.5 py-0.5 text-xs font-medium text-muted-foreground">
      {children}
    </span>
  );
}

function InvoiceTableSkeleton() {
  return (
    <Card
      variant="light"
      className="bg-transparent ring-1 ring-border/50 border-none"
    >
      <CardHeader className="p-4 border-b border-border/50">
        <Skeleton className="h-4 w-24" />
      </CardHeader>
      <CardContent className="p-4 space-y-3">
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-8 w-3/4" />
      </CardContent>
    </Card>
  );
}

function openHostedInvoice(url: string) {
  window.open(url, '_blank', 'noopener,noreferrer');
}

function sortInvoices(invoices: OrganizationInvoice[]) {
  return [...invoices].sort((a, b) => {
    const aTime = a.createdAt ? new Date(a.createdAt).getTime() : 0;
    const bTime = b.createdAt ? new Date(b.createdAt).getTime() : 0;
    return bTime - aTime;
  });
}

export function Invoices({
  invoicePreviews,
  invoices,
  isLoading,
  isError,
  onRetry,
}: InvoicesProps) {
  const [selectedPreview, setSelectedPreview] =
    useState<OrganizationInvoicePreview | null>(null);

  const upcoming = invoicePreviews ?? [];
  const previous = useMemo(() => sortInvoices(invoices ?? []), [invoices]);

  if (isLoading) {
    return <InvoiceTableSkeleton />;
  }

  if (isError) {
    return (
      <Alert variant="warn">
        <AlertTitle>Invoices unavailable</AlertTitle>
        <AlertDescription className="flex flex-col gap-3">
          <span>
            We couldn&apos;t load invoices for this organization. Your
            subscription details are still available below.
          </span>
          {onRetry ? (
            <div>
              <Button size="sm" variant="outline" onClick={onRetry}>
                Try again
              </Button>
            </div>
          ) : null}
        </AlertDescription>
      </Alert>
    );
  }

  if (upcoming.length === 0 && previous.length === 0) {
    return (
      <Card
        variant="light"
        className="bg-transparent ring-1 ring-border/50 border-none"
      >
        <CardHeader className="p-4 border-b border-border/50">
          <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground">
            Invoices
          </CardTitle>
        </CardHeader>
        <CardContent className="p-4">
          <p className="text-sm text-muted-foreground">
            No invoices yet. Upcoming and previous invoices will appear here.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {upcoming.length > 0 ? (
        <div className="space-y-2">
          <InvoiceSectionLabel>Upcoming</InvoiceSectionLabel>
          <Card
            variant="light"
            className="bg-transparent ring-1 ring-border/50 border-none"
          >
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="px-4">Products</TableHead>
                    <TableHead>Total</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Date</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {upcoming.map((preview, index) => {
                    const status = invoiceStatusBadge('upcoming');
                    return (
                      <TableRow
                        key={`${preview.planIds.join('-')}-${preview.invoiceAt}-${index}`}
                        className="cursor-pointer"
                        onClick={() => setSelectedPreview(preview)}
                      >
                        <TableCell className="px-4 font-medium text-foreground">
                          {formatPlanIds(preview.planIds)}
                        </TableCell>
                        <TableCell className="text-foreground">
                          {formatInvoiceAmount(
                            preview.totalCents,
                            preview.currency,
                          )}
                        </TableCell>
                        <TableCell>
                          <Badge variant={status.variant}>{status.label}</Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatInvoiceDate(preview.invoiceAt)}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>
      ) : null}

      {previous.length > 0 ? (
        <div className="space-y-2">
          <InvoiceSectionLabel>Previous</InvoiceSectionLabel>
          <Card
            variant="light"
            className="bg-transparent ring-1 ring-border/50 border-none"
          >
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="px-4">Products</TableHead>
                    <TableHead>Total</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Date</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {previous.map((invoice) => {
                    const status = invoiceStatusBadge(invoice.status);
                    const hostedInvoiceUrl = invoice.hostedInvoiceUrl;
                    const canOpen = Boolean(hostedInvoiceUrl);
                    return (
                      <TableRow
                        key={invoice.stripeId}
                        className={canOpen ? 'cursor-pointer' : undefined}
                        onClick={
                          hostedInvoiceUrl
                            ? () => openHostedInvoice(hostedInvoiceUrl)
                            : undefined
                        }
                      >
                        <TableCell className="px-4 font-medium text-foreground">
                          <span className="inline-flex items-center gap-2">
                            {formatPlanIds(invoice.planIds)}
                            {canOpen ? (
                              <ArrowTopRightOnSquareIcon className="h-3.5 w-3.5 text-muted-foreground" />
                            ) : null}
                          </span>
                        </TableCell>
                        <TableCell className="text-foreground">
                          {formatInvoiceAmount(
                            invoice.totalCents,
                            invoice.currency,
                          )}
                        </TableCell>
                        <TableCell>
                          <Badge variant={status.variant}>{status.label}</Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatInvoiceDate(invoice.createdAt)}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>
      ) : null}

      <UpcomingInvoiceDialog
        preview={selectedPreview}
        open={selectedPreview !== null}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedPreview(null);
          }
        }}
      />
    </div>
  );
}
