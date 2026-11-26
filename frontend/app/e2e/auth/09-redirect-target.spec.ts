import {
  completeCreateTenantOnboarding,
  expect,
  signInWithForm,
  test,
} from '../support/fixtures';
import { seededUsers } from '../support/seeded-users.generated';
import type { Page } from '@playwright/test';

// Builds a deep link into the owner's first tenant, then logs out so visiting
// it has to go through the login redirect.
async function ownerDeepLinkWhileLoggedOut(page: Page): Promise<string> {
  const loginResponse = await page.request.post('/api/v1/users/login', {
    data: {
      email: seededUsers.owner.email,
      password: seededUsers.owner.password,
    },
  });
  expect(loginResponse.ok()).toBeTruthy();

  const memberships = await page.request.get('/api/v1/users/memberships');
  expect(memberships.ok()).toBeTruthy();
  const body = (await memberships.json()) as {
    rows: { tenant: { metadata: { id: string } } }[];
  };
  const tenantId = body.rows[0].tenant.metadata.id;

  const logoutResponse = await page.request.post('/api/v1/users/logout');
  expect(logoutResponse.ok()).toBeTruthy();

  return `/tenants/${tenantId}/events`;
}

function readRedirectTarget(page: Page) {
  return page.evaluate(() => sessionStorage.getItem('hatchet:redirect_target'));
}

test.describe('Redirect target: deep link restoration', () => {
  test('restores deep link after login', async ({ page }) => {
    const deepLink = await ownerDeepLinkWhileLoggedOut(page);

    await page.goto(deepLink);
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login');

    await signInWithForm(page, 'owner');

    await expect(page).toHaveURL((url) => url.pathname === deepLink, {
      timeout: 30_000,
    });
  });

  test('restores deep link after signup and tenant creation', async ({
    page,
  }) => {
    const ts = Date.now();
    const userName = `E2E Redirect ${ts}`;
    const email = `e2e+redirect+${ts}@example.com`;
    const password = `E2e${ts}pw1`;

    const deepLink = await ownerDeepLinkWhileLoggedOut(page);

    await page.goto(deepLink);
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login');

    await page.locator('a', { hasText: 'Sign up' }).click();
    await expect(page).toHaveURL((url) => url.pathname === '/auth/register');

    await page.locator('input#name').fill(userName);
    await page.locator('input#email').fill(email);
    await page.locator('input#password').fill(password);
    await page.locator('button', { hasText: 'Create Account' }).click();

    await expect(page).toHaveURL(
      (url) => url.pathname === '/onboarding/create-tenant',
    );
    expect(await readRedirectTarget(page)).toBe(deepLink);

    await completeCreateTenantOnboarding(
      page,
      `E2ERedirect${String(ts).slice(-6)}`,
    );

    // The new user doesn't own that tenant, so the page shows "Access
    // denied", but the redirect itself fired.
    await expect(page).toHaveURL((url) => url.pathname.includes(deepLink), {
      timeout: 30_000,
    });
    expect(await readRedirectTarget(page)).toBeNull();
  });
});
