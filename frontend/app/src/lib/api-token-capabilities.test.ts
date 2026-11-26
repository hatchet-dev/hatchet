import {
  buildCatalog,
  describeTokenAccess,
  grantedCapabilities,
  toRequestedCapabilities,
} from './api-token-capabilities';
import { TokenCapability } from './api/generated/data-contracts';
import assert from 'node:assert/strict';
import { test } from 'node:test';

const rows: TokenCapability[] = [
  {
    name: 'all',
    label: 'Read and write',
    description: '',
    bundle: true,
    includes: ['worker', 'runs:read', 'runs:trigger', 'members:read'],
  },
  {
    name: 'read-only',
    label: 'Read',
    description: '',
    bundle: true,
    includes: ['runs:read'],
  },
  {
    name: 'worker',
    group: 'Processes',
    description: '',
    bundle: false,
    includes: ['runs:read', 'runs:trigger'],
  },
  {
    name: 'runs:read',
    group: 'Runs',
    description: '',
    bundle: false,
    includes: [],
  },
  {
    name: 'runs:trigger',
    group: 'Runs',
    description: '',
    bundle: false,
    includes: [],
  },
  {
    name: 'members:read',
    group: 'Tenant',
    description: '',
    bundle: false,
    includes: [],
  },
];

test('the catalog follows the server: groups in order, presets from the bundles', () => {
  const catalog = buildCatalog(rows);

  assert.deepEqual(
    catalog.groups.map((g) => g.label),
    ['Processes', 'Runs', 'Tenant'],
  );
  assert.deepEqual(catalog.read, ['runs:read']);
  assert.deepEqual(catalog.write, ['worker', 'runs:trigger', 'members:read']);

  // untouched presets are sent as bundles, edited selections as a list in display order
  assert.deepEqual(toRequestedCapabilities(catalog.selectable, catalog), [
    'all',
  ]);
  assert.deepEqual(toRequestedCapabilities(['runs:read'], catalog), [
    'read-only',
  ]);
  assert.deepEqual(
    toRequestedCapabilities(['runs:trigger', 'worker'], catalog),
    ['worker', 'runs:trigger'],
  );

  // the review step shows what a capability brings with it, not only what was ticked
  assert.deepEqual([...grantedCapabilities(['worker'], catalog)].sort(), [
    'runs:read',
    'runs:trigger',
    'worker',
  ]);
});

test('token access reads as the bundle it holds', () => {
  assert.equal(
    describeTokenAccess({ capabilities: ['all'], readOnly: false }, rows),
    'Read and write',
  );
  assert.equal(
    describeTokenAccess({ capabilities: ['read-only'], readOnly: true }, rows),
    'Read (REST only)',
  );
  assert.equal(
    describeTokenAccess(
      { capabilities: ['worker', 'runs:read'], readOnly: false },
      rows,
    ),
    'worker, runs:read',
  );
  assert.equal(
    describeTokenAccess({ readOnly: true }, rows),
    'Read (REST only)',
  );
});
