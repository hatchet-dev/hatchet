import { seededUsers, type SeededUserKey } from './seeded-users.generated';
import { test as base, expect, type Page } from '@playwright/test';

export { expect };

// Rejections the app already handles through its own error UI. Chromium
// reports a thrown `Response` by its class name, so match that as well as the
// stringified form.
function isIgnoredPageError(err: Error): boolean {
  const text = `${err.name} ${err.message}`;

  return (
    err.message === 'Response' ||
    text.includes('[object Response]') ||
    text.includes('CancelledError')
  );
}

export const test = base.extend({
  // Playwright ignores uncaught page errors by default; fail the test on them
  // so app crashes surface in e2e runs.
  page: async ({ page }, use) => {
    const pageErrors: Error[] = [];
    page.on('pageerror', (err) => {
      if (!isIgnoredPageError(err)) {
        pageErrors.push(err);
      }
    });

    await use(page);

    expect(
      pageErrors.map((err) => `${err.name}: ${err.message}`),
      'uncaught errors in the page',
    ).toEqual([]);
  },
});

const TENANT_URL = /\/tenants\/[^/]+/;

export function pathnameOf(page: Page): string {
  return new URL(page.url()).pathname;
}

export function tenantBasePath(page: Page): string {
  const match = pathnameOf(page).match(/\/tenants\/[^/]+/);
  if (!match) {
    throw new Error(`Failed to derive tenant base path from: ${page.url()}`);
  }

  return match[0];
}

export function tenantIdFromUrl(page: Page): string {
  return tenantBasePath(page).split('/')[2];
}

export async function completeCreateTenantOnboarding(
  page: Page,
  tenantName: string,
) {
  await page
    .locator('input#tenant-name')
    .filter({ visible: true })
    .first()
    .fill(tenantName);
  await page.locator('button', { hasText: 'Get started' }).click();
}

// Logs in through the API on every call instead of sharing a saved
// storageState: specs log out, and a shared session would be invalidated for
// every spec that runs after them.
export async function login(page: Page, user: SeededUserKey) {
  const { email, password } = seededUsers[user];

  const response = await page.request.post('/api/v1/users/login', {
    data: { email, password },
  });
  expect(response.status()).toBe(200);

  await page.goto('/');
  await expect(page).toHaveURL(
    /\/tenants\/|\/onboarding\/create-tenant|\/onboarding\/invites/,
    { timeout: 30_000 },
  );

  const pathname = pathnameOf(page);

  // Seeded users normally have a tenant, but the suite also runs against
  // databases where they don't.
  if (pathname.includes('/onboarding/create-tenant')) {
    await completeCreateTenantOnboarding(
      page,
      `E2ESeedTenant${String(Date.now()).slice(-6)}`,
    );
  }

  if (pathname.includes('/onboarding/invites')) {
    const accepted = page.waitForResponse(
      (r) =>
        r.request().method() === 'POST' &&
        r.url().includes('/api/v1/users/invites/accept'),
    );
    await page.locator('button', { hasText: 'Accept' }).first().click();
    expect((await accepted).status()).toBe(200);
  }

  await expect(page).toHaveURL(/\/tenants\//, { timeout: 30_000 });
}

export async function signInWithForm(page: Page, user: SeededUserKey) {
  const { email, password } = seededUsers[user];

  await page.locator('input#email').fill(email);
  await page.locator('input#password').fill(password);

  const submit = page
    .locator('form')
    .filter({ visible: true })
    .first()
    .locator('button', { hasText: /^Sign In$/ });
  await expect(submit).toBeEnabled();
  await submit.click();
}

export async function createTenantViaApi(
  page: Page,
  tenant: { name: string; slug: string },
): Promise<string> {
  const response = await page.request.post('/api/v1/tenants', {
    data: { ...tenant, environment: 'development' },
  });
  expect(response.status()).toBe(200);

  const body = (await response.json()) as { metadata: { id: string } };
  return body.metadata.id;
}

export function tenantSwitcherButton(page: Page) {
  return page
    .locator('button[aria-label="Select a tenant"]')
    .filter({ visible: true })
    .first();
}

export async function switchTenant(page: Page, tenantSlug: string) {
  const switcher = tenantSwitcherButton(page);
  await expect(switcher).toBeEnabled();
  await switcher.click();

  await expect(page.getByTestId('tenant-switcher-list')).toBeVisible();
  await page.getByTestId(`tenant-switcher-item-${tenantSlug}`).click();
}

export async function logOutViaAccountMenu(page: Page) {
  await page
    .locator('button[aria-label="Open account menu"]')
    .filter({ visible: true })
    .first()
    .click();
  await page.getByText('Log out').filter({ visible: true }).first().click();
}

export async function waitForTenantShell(page: Page) {
  await expect(page).toHaveURL(TENANT_URL, { timeout: 30_000 });
}

// Declines every invite left in the open invites dialog. The modal re-renders
// after each refetch, so wait for it before looking for the next button.
export async function declineRemainingInvites(
  page: Page,
  opts: { expectRejectStatus?: number } = {},
) {
  const declineButtons = page.locator(
    '[role="dialog"][data-state="open"] button[aria-label="Decline"]',
  );

  for (let remaining = 20; remaining > 0; remaining--) {
    if ((await declineButtons.count()) === 0) {
      return;
    }

    const rejected = page.waitForResponse(
      (r) =>
        r.request().method() === 'POST' &&
        r.url().includes('/api/v1/users/invites/reject'),
    );
    const refetched = page.waitForResponse(
      (r) =>
        r.request().method() === 'GET' &&
        new URL(r.url()).pathname.startsWith('/api/v1/users/invites'),
    );

    await declineButtons.first().click();

    const rejectResponse = await rejected;
    if (opts.expectRejectStatus !== undefined) {
      expect(rejectResponse.status()).toBe(opts.expectRejectStatus);
    }
    await refetched;
  }
}
