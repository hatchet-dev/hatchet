import {
  dailyMeterSeverity,
  formatTimeUntil,
  nextRefillAt,
  selectDailyMeters,
  toUsageDisplayRows,
} from './usage-features';
import {
  OrganizationTenantResourceLimits,
  OrganizationUsageFeature,
  TenantResource,
  TenantResourceLimit,
} from '@/lib/api/generated/control-plane/data-contracts';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

function feature(
  featureId: string,
  overrides: Partial<OrganizationUsageFeature> = {},
): OrganizationUsageFeature {
  return {
    featureId,
    name: featureId,
    usage: 0,
    includedUsage: 0,
    unlimited: false,
    ...overrides,
  };
}

function limit(
  resource: TenantResource,
  overrides: Partial<TenantResourceLimit> = {},
): TenantResourceLimit {
  return {
    metadata: {
      id: `${resource}-limit`,
      createdAt: '2026-09-09T00:00:00.000Z',
      updatedAt: '2026-09-09T00:00:00.000Z',
    },
    resource,
    limitValue: 2000,
    value: 0,
    window: '24h0m0s',
    lastRefill: '2026-09-09T00:00:00.000Z',
    ...overrides,
  };
}

function tenant(
  tenantId: string,
  tenantName: string,
  limits: TenantResourceLimit[],
): OrganizationTenantResourceLimits {
  return {
    tenantId,
    tenantName,
    tenantSlug: tenantName,
    limits,
  };
}

describe('toUsageDisplayRows', () => {
  it('hides autumn daily-limit rows and attaches the live meter', () => {
    const rows = toUsageDisplayRows(
      [
        feature('users', { name: 'Users', usage: 1, includedUsage: 1 }),
        feature('task_runs', {
          name: 'Task Runs',
          usage: 12,
          includedUsage: 2000,
        }),
        feature('events', {
          name: 'External Events',
          usage: 4,
          includedUsage: 1000,
        }),
        feature('task_runs_daily_limit', {
          name: 'Task Runs Daily Limit',
          includedUsage: 2000,
        }),
        feature('events_daily_limit', {
          name: 'External Events Daily Limit',
          includedUsage: 1000,
        }),
      ],
      {
        task_runs: {
          ...limit(TenantResource.TASK_RUN, { value: 75, limitValue: 2000 }),
          tenantName: 'asdf',
        },
      },
    );

    assert.deepEqual(
      rows.map((row) => ({
        id: row.feature.featureId,
        periodUsage: row.feature.usage,
        showPeriodAsCount: row.showPeriodAsCount,
        meterValue: row.dailyMeter?.value,
        meterLimit: row.dailyMeter?.limitValue,
      })),
      [
        {
          id: 'users',
          periodUsage: 1,
          showPeriodAsCount: false,
          meterValue: undefined,
          meterLimit: undefined,
        },
        {
          id: 'task_runs',
          periodUsage: 12,
          showPeriodAsCount: true,
          meterValue: 75,
          meterLimit: 2000,
        },
        {
          id: 'events',
          periodUsage: 4,
          showPeriodAsCount: true,
          meterValue: undefined,
          meterLimit: undefined,
        },
      ],
    );
  });

  it('keeps period rows as counts when there is no live daily meter', () => {
    const rows = toUsageDisplayRows([
      feature('task_runs', {
        name: 'Task Runs',
        usage: 250,
        includedUsage: 100000,
      }),
      feature('events', { name: 'External Events', usage: 8, unlimited: true }),
    ]);

    assert.equal(rows.length, 2);
    assert.equal(rows[0].dailyMeter, undefined);
    assert.equal(rows[0].showPeriodAsCount, true);
    assert.equal(rows[1].dailyMeter, undefined);
  });
});

describe('selectDailyMeters', () => {
  it('uses the selected tenant meter', () => {
    const meters = selectDailyMeters(
      [
        tenant('a', 'Alpha', [
          limit(TenantResource.TASK_RUN, { value: 10, limitValue: 2000 }),
        ]),
        tenant('b', 'Beta', [
          limit(TenantResource.TASK_RUN, { value: 900, limitValue: 2000 }),
        ]),
      ],
      'a',
    );

    assert.equal(meters.task_runs?.value, 10);
    assert.equal(meters.task_runs?.showTenant, false);
  });

  it('picks the most consumed tenant when viewing all tenants', () => {
    const meters = selectDailyMeters(
      [
        tenant('a', 'Alpha', [
          limit(TenantResource.TASK_RUN, { value: 10, limitValue: 2000 }),
          limit(TenantResource.EVENT, { value: 900, limitValue: 1000 }),
        ]),
        tenant('b', 'Beta', [
          limit(TenantResource.TASK_RUN, { value: 1800, limitValue: 2000 }),
          limit(TenantResource.EVENT, { value: 10, limitValue: 1000 }),
        ]),
      ],
      'all',
    );

    assert.equal(meters.task_runs?.value, 1800);
    assert.equal(meters.task_runs?.tenantName, 'Beta');
    assert.equal(meters.task_runs?.showTenant, true);
    assert.equal(meters.events?.value, 900);
    assert.equal(meters.events?.tenantName, 'Alpha');
  });
});

describe('dailyMeterSeverity', () => {
  it('warns at 75% and goes critical at 90%', () => {
    assert.equal(dailyMeterSeverity({ value: 1499, limitValue: 2000 }), 'ok');
    assert.equal(dailyMeterSeverity({ value: 1500, limitValue: 2000 }), 'warn');
    assert.equal(
      dailyMeterSeverity({ value: 1800, limitValue: 2000 }),
      'critical',
    );
    assert.equal(
      dailyMeterSeverity({ value: 2000, limitValue: 2000 }),
      'critical',
    );
  });

  it('honors an earlier alarm threshold', () => {
    assert.equal(
      dailyMeterSeverity({ value: 600, limitValue: 1000, alarmValue: 500 }),
      'warn',
    );
  });
});

describe('nextRefillAt', () => {
  it('adds the meter window to the last refill', () => {
    const next = nextRefillAt({
      lastRefill: '2026-09-09T00:00:00.000Z',
      window: '24h0m0s',
    });

    assert.equal(next?.toISOString(), '2026-09-10T00:00:00.000Z');
  });

  it('formats a compact refill countdown', () => {
    const now = Date.parse('2026-09-09T10:00:00.000Z');
    assert.equal(
      formatTimeUntil(new Date('2026-09-09T10:20:00.000Z'), now),
      'refills in 20m',
    );
    assert.equal(
      formatTimeUntil(new Date('2026-09-09T16:00:00.000Z'), now),
      'refills in 6h',
    );
  });
});
