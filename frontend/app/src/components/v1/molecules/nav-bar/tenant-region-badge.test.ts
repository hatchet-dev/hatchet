import {
  formatTenantDeploymentBadgeLabel,
  formatTenantDeploymentBadgeTooltip,
  formatTenantRegionDisplay,
} from './tenant-region-badge';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

describe('formatTenantRegionDisplay', () => {
  it('strips provider prefix', () => {
    assert.equal(formatTenantRegionDisplay('aws:us-east-1'), 'us-east-1');
  });
});

describe('formatTenantDeploymentBadgeLabel', () => {
  it('prefers shard display name', () => {
    assert.equal(
      formatTenantDeploymentBadgeLabel({
        region: 'aws:us-east-1',
        shardDisplayName: 'Enter',
      }),
      'Enter',
    );
  });

  it('falls back to stripped region', () => {
    assert.equal(
      formatTenantDeploymentBadgeLabel({ region: 'aws:us-east-1' }),
      'us-east-1',
    );
  });
});

describe('formatTenantDeploymentBadgeTooltip', () => {
  it('includes both label and region when display name is set', () => {
    assert.equal(
      formatTenantDeploymentBadgeTooltip({
        region: 'aws:us-east-1',
        shardDisplayName: 'Enter',
      }),
      'Enter (aws:us-east-1)',
    );
  });
});
