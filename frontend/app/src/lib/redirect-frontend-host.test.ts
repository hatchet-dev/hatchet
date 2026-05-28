import {
  buildRedirectFrontendHref,
  domainRedirectSkipStorageKey,
  parseRedirectFrontendOrigin,
} from './redirect-frontend-host';
import assert from 'node:assert/strict';
import { test } from 'node:test';

test('parseRedirectFrontendOrigin accepts full URL', () => {
  const u = parseRedirectFrontendOrigin('https://cloud.hatchet.run');
  assert.ok(u);
  assert.equal(u.origin, 'https://cloud.hatchet.run');
});

test('parseRedirectFrontendOrigin defaults host-only to https', () => {
  const u = parseRedirectFrontendOrigin('cloud.hatchet.run');
  assert.ok(u);
  assert.equal(u.origin, 'https://cloud.hatchet.run');
});

test('parseRedirectFrontendOrigin returns null for empty', () => {
  assert.equal(parseRedirectFrontendOrigin(''), null);
  assert.equal(parseRedirectFrontendOrigin('   '), null);
});

test('buildRedirectFrontendHref preserves path query and hash', () => {
  const origin = parseRedirectFrontendOrigin('https://cloud.hatchet.run');
  assert.ok(origin);
  const href = buildRedirectFrontendHref(origin, {
    pathname: '/t/foo/runs',
    search: '?q=1',
    hash: '#a',
  });
  assert.equal(href, 'https://cloud.hatchet.run/t/foo/runs?q=1#a');
});

test('domainRedirectSkipStorageKey is stable per origin', () => {
  assert.equal(
    domainRedirectSkipStorageKey('https://cloud.hatchet.run'),
    'hatchet:domain-redirect-skip:https://cloud.hatchet.run',
  );
});
