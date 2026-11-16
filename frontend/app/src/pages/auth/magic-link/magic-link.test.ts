import {
  accountSwitchMessage,
  isTerminalMagicLinkError,
  showsMagicLinkLogin,
  takeFragmentToken,
} from './magic-link';
import * as assert from 'node:assert/strict';
import { describe, test } from 'node:test';

describe('magic link login visibility', () => {
  test('stays off for the open-source login path', () => {
    assert.equal(showsMagicLinkLogin(false, ['magic_link', 'basic']), false);
    assert.equal(showsMagicLinkLogin(false, undefined), false);
  });

  test('appears only when the control plane advertises it', () => {
    assert.equal(showsMagicLinkLogin(true, ['basic', 'google']), false);
    assert.equal(showsMagicLinkLogin(true, ['magic_link']), true);
  });
});

describe('magic link confirmation', () => {
  test('reads the token from the fragment and drops it from the URL', () => {
    const taken = takeFragmentToken(
      'https://cloud.example/auth/magic-link?next=%2F#one-time-token',
    );

    assert.equal(taken.token, 'one-time-token');
    assert.equal(taken.urlWithoutFragment, '/auth/magic-link?next=%2F');
    assert.equal(taken.urlWithoutFragment.includes('#'), false);
    assert.equal(taken.urlWithoutFragment.includes('one-time-token'), false);
  });

  test('drops a fragment that is not a valid token', () => {
    const taken = takeFragmentToken(
      'https://cloud.example/auth/magic-link#%E0%A4%A',
    );

    assert.equal(taken.token, null);
    assert.equal(taken.urlWithoutFragment, '/auth/magic-link');
  });

  test('treats a missing fragment as an unusable link', () => {
    const taken = takeFragmentToken('https://cloud.example/auth/magic-link');

    assert.equal(taken.token, null);
    assert.equal(taken.urlWithoutFragment, '/auth/magic-link');
  });

  test('warns before replacing another signed-in account', () => {
    assert.equal(accountSwitchMessage(null), null);
    assert.equal(accountSwitchMessage(''), null);
    assert.match(
      accountSwitchMessage('owner@example.com') ?? '',
      /owner@example.com/,
    );
    assert.match(
      accountSwitchMessage('owner@example.com') ?? '',
      /replaces this browser session/,
    );
  });

  test('only rejected or disabled links end the confirmation', () => {
    assert.equal(isTerminalMagicLinkError(400), true);
    assert.equal(isTerminalMagicLinkError(405), true);
    assert.equal(isTerminalMagicLinkError(429), false);
    assert.equal(isTerminalMagicLinkError(500), false);
    assert.equal(isTerminalMagicLinkError(undefined), false);
  });
});
