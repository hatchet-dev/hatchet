import {
  formatShardDeploymentKey,
  shardDeploymentKey,
} from './shard-deployment-key';
import {
  OrganizationAvailableShard,
  OrganizationAvailableShardClass,
} from '@/lib/api/generated/control-plane/data-contracts';
import * as assert from 'node:assert';
import { describe, test } from 'node:test';

const sharedShard = {
  provider: 'aws',
  region: 'us-west-2',
  shardClass: OrganizationAvailableShardClass.SHARED,
} as const;

const namedShard = {
  ...sharedShard,
  shardName: 'foo',
} as const;

const dedicatedShard = {
  ...sharedShard,
  shardClass: OrganizationAvailableShardClass.DEDICATED,
} as const;

const dedicatedNamedShard = {
  ...dedicatedShard,
  shardName: 'foo',
} as const;

describe('shard-deployment-key rendering', () => {
  test('shardDeploymentKey', () => {
    const cases: Array<{
      name: string;
      shard: OrganizationAvailableShard;
      want: string;
    }> = [
      {
        name: 'existing unnamed shared shard',
        shard: sharedShard,
        want: 'aws:us-west-2',
      },
      {
        name: 'named shared shard',
        shard: namedShard,
        want: 'aws:us-west-2:foo',
      },
      {
        name: 'existing unnamed dedicated shard',
        shard: dedicatedShard,
        want: 'aws:us-west-2',
      },
      {
        name: 'named dedicated shard',
        shard: dedicatedNamedShard,
        want: 'aws:us-west-2:foo',
      },
      {
        name: 'providerless fallback row',
        shard: { ...sharedShard, provider: '' },
        want: 'us-west-2',
      },
    ];

    for (const tc of cases) {
      assert.strictEqual(shardDeploymentKey(tc.shard), tc.want, tc.name);
    }
  });

  test('formatShardDeploymentKey', () => {
    const cases: Array<{
      name: string;
      region: string | undefined;
      want: string | undefined;
    }> = [
      {
        name: 'existing unnamed compact region',
        region: 'aws:us-west-2',
        want: 'us-west-2',
      },
      {
        name: 'named compact region',
        region: 'aws:us-west-2:west',
        want: 'us-west-2 / west',
      },
      {
        name: 'region-only fallback',
        region: 'us-west-2',
        want: 'us-west-2',
      },
      {
        name: 'missing region',
        region: undefined,
        want: undefined,
      },
    ];

    for (const tc of cases) {
      assert.strictEqual(formatShardDeploymentKey(tc.region), tc.want, tc.name);
    }
  });
});
