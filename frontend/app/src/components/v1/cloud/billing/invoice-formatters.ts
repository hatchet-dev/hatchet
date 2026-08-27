import { BadgeProps } from '@/components/v1/ui/badge';

export function formatInvoiceAmount(cents: number, currency = 'usd'): string {
  try {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency.toUpperCase(),
    }).format(cents / 100);
  } catch {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(cents / 100);
  }
}

export function formatInvoiceDate(value?: string) {
  if (!value) {
    return '—';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '—';
  }
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

export function formatInvoicePeriod(start?: string, end?: string) {
  const startLabel = formatInvoiceDate(start);
  const endLabel = formatInvoiceDate(end);
  if (startLabel === '—' && endLabel === '—') {
    return null;
  }
  if (startLabel === '—') {
    return endLabel;
  }
  if (endLabel === '—') {
    return startLabel;
  }
  return `${startLabel} – ${endLabel}`;
}

export function formatPlanIds(planIds?: string[]) {
  if (!planIds || planIds.length === 0) {
    return 'Invoice';
  }

  return planIds
    .map((planId) =>
      planId
        .split('_')
        .filter(Boolean)
        .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
        .join(' '),
    )
    .join(', ');
}

export function formatQuantity(quantity: number) {
  return new Intl.NumberFormat('en-US').format(quantity);
}

export type InvoiceStatusTone = {
  label: string;
  variant: BadgeProps['variant'];
};

export function invoiceStatusBadge(status?: string): InvoiceStatusTone {
  switch ((status ?? '').toLowerCase()) {
    case 'paid':
      return { label: 'Paid', variant: 'successful' };
    case 'open':
      return { label: 'Open', variant: 'inProgress' };
    case 'draft':
      return { label: 'Draft', variant: 'queued' };
    case 'upcoming':
      return { label: 'Upcoming', variant: 'inProgress' };
    case 'void':
      return { label: 'Void', variant: 'cancelled' };
    case 'uncollectible':
      return { label: 'Uncollectible', variant: 'failed' };
    default:
      return {
        label: status
          ? status.charAt(0).toUpperCase() + status.slice(1)
          : 'Unknown',
        variant: 'queued',
      };
  }
}
