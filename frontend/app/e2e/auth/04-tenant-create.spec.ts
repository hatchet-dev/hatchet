import {
  expect,
  login,
  TAGS,
  tenantSwitcherButton,
  test,
} from '../support/fixtures';

test.describe('Tenants: create', { tag: TAGS.seeded }, () => {
  test('creates a new tenant', async ({ page }) => {
    const ts = Date.now();
    await login(page, 'owner');
    await page.goto('/');

    await tenantSwitcherButton(page).click();
    await page.getByTestId('new-tenant').click();

    const dialog = page.locator('[role="dialog"]');
    await expect(dialog).toBeVisible({ timeout: 5_000 });

    const tenantName = `E2ETenant${String(ts).slice(-6)}`;
    await dialog
      .locator('#tenant-name')
      .filter({ visible: true })
      .first()
      .fill(tenantName);

    const created = page.waitForResponse(
      (r) =>
        r.request().method() === 'POST' &&
        new URL(r.url()).pathname === '/api/v1/tenants',
    );
    await dialog.locator('button[type="submit"]').click();
    expect((await created).status()).toBe(200);

    await expect(page).toHaveURL(/\/tenants\/.+\/overview/, {
      timeout: 30_000,
    });
    await expect(page.locator('h1', { hasText: 'Overview' })).toBeVisible({
      timeout: 30_000,
    });
    await expect(tenantSwitcherButton(page)).toContainText(tenantName);
  });
});
