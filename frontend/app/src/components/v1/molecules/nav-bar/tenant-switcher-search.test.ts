import {
  getHighlightRanges,
  matchesSearch,
  normalizeSearchText,
  parseSearchQuery,
} from './tenant-switcher-search';
import { formatShardDeploymentKey } from '@/lib/shard-deployment-key';
import * as assert from 'node:assert';
import { test } from 'node:test';

test('normalizeSearchText ignores case and separators', () => {
  assert.equal(
    normalizeSearchText('Customer-Success_Sandbox.EU'),
    'customersuccesssandboxeu',
  );
  assert.equal(normalizeSearchText('aws:us-west-2 / west'), 'awsuswest2west');
});

test('parseSearchQuery splits on whitespace and drops empty words', () => {
  assert.deepEqual(parseSearchQuery('  Prod   EU-west '), ['prod', 'euwest']);
  assert.deepEqual(parseSearchQuery(' - '), []);
});

test('matchesSearch ignores dashes and capitals', () => {
  assert.ok(matchesSearch(parseSearchQuery('controlplane'), ['control-plane']));
  assert.ok(
    matchesSearch(parseSearchQuery('CONTROL_PLANE'), ['control-plane']),
  );
});

test('matchesSearch accepts words in any order', () => {
  assert.ok(matchesSearch(parseSearchQuery('eu prod'), ['production-eu']));
});

test('matchesSearch requires every word, across any field', () => {
  const fields = ['staging', 'staging', 'us-east-1', 'hatchet.run'];
  assert.ok(matchesSearch(parseSearchQuery('staging hatchet'), fields));
  assert.ok(matchesSearch(parseSearchQuery('useast'), fields));
  assert.ok(!matchesSearch(parseSearchQuery('staging eu'), fields));
});

test('matchesSearch matches everything for an empty query', () => {
  assert.ok(matchesSearch([], ['anything']));
});

test('getHighlightRanges maps normalized matches back to the original text', () => {
  // "controlpl" spans the dash in "control-plane".
  assert.deepEqual(getHighlightRanges('control-plane', ['controlpl']), [
    { start: 0, end: 10 },
  ]);
});

test('getHighlightRanges highlights each word and merges overlaps', () => {
  assert.deepEqual(getHighlightRanges('production-eu', ['eu', 'prod']), [
    { start: 0, end: 4 },
    { start: 11, end: 13 },
  ]);
  assert.deepEqual(getHighlightRanges('production', ['prod', 'rodu']), [
    { start: 0, end: 5 },
  ]);
});

test('getHighlightRanges finds repeated matches', () => {
  assert.deepEqual(getHighlightRanges('sdk-ci-sdk', ['sdk']), [
    { start: 0, end: 3 },
    { start: 7, end: 10 },
  ]);
});

test('getHighlightRanges returns nothing for words that only match other fields', () => {
  assert.deepEqual(getHighlightRanges('staging', ['hatchet']), []);
});

test('matchesSearch finds tenants by region and shard name', () => {
  const region = 'aws:us-west-2:dedicated-shard';
  const fields = [
    'sandbox',
    'sandbox',
    region,
    formatShardDeploymentKey(region),
  ];
  assert.ok(matchesSearch(parseSearchQuery('dedicated-shard'), fields));
  assert.ok(matchesSearch(parseSearchQuery('us-west-2 / dedicated'), fields));
  assert.ok(matchesSearch(parseSearchQuery('shard west2'), fields));
  assert.ok(!matchesSearch(parseSearchQuery('us-east-1'), fields));
});
