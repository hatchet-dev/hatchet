import { gotoStory, storyActionCalls } from '../support/storybook';
import { expect, TAGS, test } from '../support/test';
import type { Page } from '@playwright/test';

const STORY_TITLE = 'Molecules/TenantSwitcherPanel';
const DEFAULT_STORY = 'molecules-tenantswitcherpanel--default';

type IndexEntry = { id: string; title: string; type: string };

async function selectedTenantSlugs(page: Page, argName: string) {
  const calls = await storyActionCalls(page, argName);
  return calls.map(([tenant]) => (tenant as { slug: string }).slug);
}

function organizationHeader(page: Page, name: string) {
  return page.locator('[role="option"][aria-expanded]', { hasText: name });
}

test.describe('TenantSwitcherPanel', { tag: TAGS.storybook }, () => {
  // Play functions only run when a story is opened, so this keeps them from
  // rotting unnoticed.
  test('every story renders and its play function passes', async ({
    page,
    request,
  }) => {
    const index = await (await request.get('/index.json')).json();
    const storyIds = (Object.values(index.entries) as IndexEntry[])
      .filter((entry) => entry.type === 'story' && entry.title === STORY_TITLE)
      .map((entry) => entry.id);
    expect(storyIds).toContain(DEFAULT_STORY);

    for (const storyId of storyIds) {
      await test.step(storyId, () => gotoStory(page, storyId));
    }
  });

  test.describe('keyboard', () => {
    test.beforeEach(async ({ page }) => {
      await gotoStory(page, DEFAULT_STORY);
      await expect(page.getByRole('combobox')).toBeFocused();
    });

    test('Enter goes back to the most recent tenant', async ({ page }) => {
      await page.keyboard.press('Enter');

      await expect
        .poll(() => selectedTenantSlugs(page, 'onSelectTenant'))
        .toEqual(['production']);
    });

    test('Enter while searching picks the first match', async ({ page }) => {
      await page.keyboard.type('prod eu');
      await expect(page.getByText('2 matches')).toBeVisible();
      await page.keyboard.press('Enter');

      await expect
        .poll(() => selectedTenantSlugs(page, 'onSelectTenant'))
        .toEqual(['prod-eu']);
    });

    test('arrow keys walk the organization tree', async ({ page }) => {
      const currentOrg = organizationHeader(page, 'Current org');

      // Past the three recent tenants to the current organization.
      for (let i = 0; i < 3; i++) {
        await page.keyboard.press('ArrowDown');
      }
      await expect(currentOrg).toHaveAttribute('aria-selected', 'true');

      await page.keyboard.press('ArrowLeft');
      await expect(currentOrg).toHaveAttribute('aria-expanded', 'false');
      await page.keyboard.press('ArrowRight');
      await expect(currentOrg).toHaveAttribute('aria-expanded', 'true');

      // From one of its tenants, ← goes back up to the organization.
      await page.keyboard.press('ArrowDown');
      await expect(
        page.getByTestId('tenant-switcher-item-control-plane'),
      ).toHaveAttribute('aria-selected', 'true');
      await page.keyboard.press('ArrowLeft');
      await expect(currentOrg).toHaveAttribute('aria-selected', 'true');
    });

    test('Cmd/Ctrl+Enter opens the tenant in a new tab', async ({
      page,
      context,
    }) => {
      const popup = context.waitForEvent('page');
      await page.keyboard.press('ControlOrMeta+Enter');

      await expect(await popup).toHaveURL(
        /\/tenants\/tenant-production\/runs$/,
      );
      expect(await storyActionCalls(page, 'onSelectTenant')).toEqual([]);
    });
  });

  test.describe('pointer', () => {
    test.beforeEach(({ page }) => gotoStory(page, DEFAULT_STORY));

    test('Show more reveals the rest of a large organization', async ({
      page,
    }) => {
      const rows = page.locator('[data-testid^="tenant-switcher-item-"]');
      const visibleBefore = await rows.count();

      await page.getByRole('option', { name: /^Show \d+ more$/ }).click();

      // Current org has 11 tenants and starts with 6 of them.
      await expect(rows).toHaveCount(visibleBefore + 5);
      await expect(
        page.getByTestId('tenant-switcher-item-sdk-ci-ts'),
      ).toBeVisible();
    });

    test('the row menu opens settings without selecting the tenant', async ({
      page,
    }) => {
      const row = page.getByTestId('tenant-switcher-item-development');
      await row.hover();
      await row
        .getByRole('button', { name: 'More actions for development' })
        .click();
      await page.getByRole('menuitem', { name: 'Tenant settings' }).click();

      await expect
        .poll(() => selectedTenantSlugs(page, 'onTenantSettings'))
        .toEqual(['development']);
      expect(await storyActionCalls(page, 'onSelectTenant')).toEqual([]);
      // Focus goes back to the search so keyboard navigation can continue.
      await expect(page.getByRole('combobox')).toBeFocused();
    });

    test('the row menu copies the tenant ID', async ({ page, context }) => {
      await context.grantPermissions(['clipboard-read', 'clipboard-write']);

      const row = page.getByTestId('tenant-switcher-item-development');
      await row.hover();
      await row
        .getByRole('button', { name: 'More actions for development' })
        .click();
      await page.getByRole('menuitem', { name: 'Copy tenant ID' }).click();

      await expect
        .poll(() => page.evaluate(() => navigator.clipboard.readText()))
        .toBe('tenant-development');
    });
  });
});
