import {
  formatInvoiceAmount,
  formatInvoiceDate,
  formatPlanIds,
  invoiceStatusBadge,
} from './invoice-formatters';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

describe('formatInvoiceAmount', () => {
  it('formats cents in the given currency', () => {
    assert.equal(formatInvoiceAmount(51000, 'usd'), '$510.00');
  });

  it('falls back to USD for invalid currencies', () => {
    assert.equal(formatInvoiceAmount(100, 'not-a-currency'), '$1.00');
  });
});

describe('formatInvoiceDate', () => {
  it('formats a valid timestamp', () => {
    assert.notEqual(formatInvoiceDate('2026-07-30T16:31:00.000Z'), '—');
  });

  it('returns an em dash for missing or invalid values', () => {
    assert.equal(formatInvoiceDate(), '—');
    assert.equal(formatInvoiceDate('not-a-date'), '—');
  });
});

describe('formatPlanIds', () => {
  it('title-cases plan ids', () => {
    assert.equal(formatPlanIds(['team_monthly']), 'Team Monthly');
  });

  it('joins multiple plan ids', () => {
    assert.equal(
      formatPlanIds(['team_monthly', 'task_runs']),
      'Team Monthly, Task Runs',
    );
  });

  it('uses a fallback when no plan ids are present', () => {
    assert.equal(formatPlanIds([]), 'Invoice');
    assert.equal(formatPlanIds(undefined), 'Invoice');
  });
});

describe('invoiceStatusBadge', () => {
  it('maps known statuses to badge variants', () => {
    assert.deepEqual(invoiceStatusBadge('paid'), {
      label: 'Paid',
      variant: 'successful',
    });
    assert.deepEqual(invoiceStatusBadge('upcoming'), {
      label: 'Upcoming',
      variant: 'inProgress',
    });
    assert.deepEqual(invoiceStatusBadge('void'), {
      label: 'Void',
      variant: 'cancelled',
    });
  });
});
