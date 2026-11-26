import type {
  APIErrors,
  APIMeta,
} from '../../src/lib/api/generated/data-contracts';
import type { Page } from '@playwright/test';

// Same responses a self-hosted API with default settings returns, which is
// what the @seeded project runs against.
export async function mockSelfHostedApi(page: Page) {
  // A 404 is how the app detects a self-hosted install, which keeps requests
  // on the same origin instead of exchanging tokens for a shard.
  await page.route(
    (url) => url.pathname === '/api/v1/control-plane/metadata',
    (route) => route.fulfill({ status: 404, json: {} }),
  );

  // The OSS server answers 200 with an "oss" error instead of cloud metadata.
  await page.route(
    (url) => url.pathname === '/api/v1/cloud/metadata',
    (route) =>
      route.fulfill({
        json: { errors: [{ description: 'oss' }] } satisfies APIErrors,
      }),
  );

  await page.route(
    (url) => url.pathname === '/api/v1/meta',
    (route) =>
      route.fulfill({
        json: {
          auth: { schemes: ['basic'] },
          allowSignup: true,
          allowInvites: true,
          allowCreateTenant: true,
          allowChangePassword: true,
        } satisfies APIMeta,
      }),
  );
}

export async function mockLoggedOutUser(page: Page) {
  await page.route(
    (url) =>
      url.pathname === '/api/v1/users/current' ||
      url.pathname === '/api/v1/users/memberships',
    (route) =>
      route.fulfill({
        status: 401,
        json: {
          errors: [{ description: 'Not authorized to view this resource' }],
        } satisfies APIErrors,
      }),
  );
}
