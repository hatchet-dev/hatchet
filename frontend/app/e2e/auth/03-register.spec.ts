import { expect, TAGS, test } from '../support/fixtures';

test.describe('Onboarding: create tenant', { tag: TAGS.seeded }, () => {
  test('registers a new user and creates a tenant', async ({ page }) => {
    const ts = Date.now();
    const userName = `E2E User ${ts}`;
    const email = `e2e+${ts}@example.com`;
    // Password must include upper+lower+number.
    const password = `E2e${ts}pw1`;

    await page.goto('/auth/register');

    await page.locator('input#name').fill(userName);
    await page.locator('input#email').fill(email);
    await page.locator('input#password').fill(password);

    const registered = page.waitForResponse(
      (r) =>
        r.request().method() === 'POST' &&
        r.url().includes('/api/v1/users/register'),
    );
    await page.locator('button', { hasText: 'Create Account' }).click();
    expect((await registered).status()).toBe(200);

    await expect(page).toHaveURL(/\/onboarding\/create-tenant/);
    await expect(
      page.locator('h2', { hasText: 'Create a new tenant' }),
    ).toBeVisible({ timeout: 30_000 });
  });
});
