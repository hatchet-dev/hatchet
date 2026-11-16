import {
  deriveEndpointUrls,
  generateSigningSecret,
  validateEndpointUrl,
} from './platforms';
import { V1ServerlessEndpointKind } from '@/lib/api/generated/data-contracts';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

describe('deriveEndpointUrls', () => {
  it('appends the adapter paths and drops trailing slashes', () => {
    assert.deepEqual(
      deriveEndpointUrls(
        V1ServerlessEndpointKind.VERCEL,
        ' https://app.vercel.app/ ',
      ),
      {
        healthcheckUrl: 'https://app.vercel.app/api/hatchet/healthcheck',
        triggerUrl: 'https://app.vercel.app/api/hatchet/trigger',
      },
    );
    assert.equal(
      deriveEndpointUrls(
        V1ServerlessEndpointKind.CLOUDFLARE_WORKERS,
        'https://w.acme.workers.dev',
      ).triggerUrl,
      'https://w.acme.workers.dev/hatchet/trigger',
    );
  });

  it('derives nothing for generic endpoints', () => {
    assert.deepEqual(
      deriveEndpointUrls(V1ServerlessEndpointKind.GENERIC_HTTP, 'https://x'),
      { healthcheckUrl: '', triggerUrl: '' },
    );
  });
});

describe('validateEndpointUrl', () => {
  it('accepts https on the default port', () => {
    assert.equal(validateEndpointUrl('https://a.example/h'), undefined);
    assert.equal(validateEndpointUrl('https://a.example:443/h'), undefined);
  });

  it('mirrors the server rules', () => {
    assert.ok(validateEndpointUrl('http://a.example'));
    assert.ok(validateEndpointUrl('https://a.example:8443'));
    assert.ok(validateEndpointUrl('https://u:p@a.example'));
    assert.ok(validateEndpointUrl('not a url'));
  });
});

describe('generateSigningSecret', () => {
  it('produces 64 hex characters', () => {
    assert.match(generateSigningSecret(), /^[0-9a-f]{64}$/);
  });
});
