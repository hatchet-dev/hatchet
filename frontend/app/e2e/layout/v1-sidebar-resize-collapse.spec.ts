import { expect, login, test } from '../support/fixtures';
import type { Page } from '@playwright/test';

const DEFAULT_EXPANDED_WIDTH = 200;
const COLLAPSED_WIDTH = 56;

const DESKTOP = { width: 1280, height: 800 };
const MOBILE = { width: 375, height: 667 };

async function visitAuthed(
  page: Page,
  viewport: { width: number; height: number },
  opts?: { requireVisibleSidebar?: boolean },
) {
  await page.setViewportSize(viewport);
  await login(page, 'owner');
  // Start each test from a clean sidebar state (but keep auth cookies).
  await page.evaluate(() => {
    localStorage.removeItem('v1SidebarCollapsed');
    localStorage.removeItem('v1SidebarWidthExpanded');
    localStorage.removeItem('v1SidebarWidth'); // legacy key
  });
  await page.goto('/');

  if (opts?.requireVisibleSidebar !== false) {
    if (viewport.width < 768) {
      await page
        .locator('button[aria-label="Toggle sidebar"]')
        .click({ timeout: 30_000 });
    }
    await expect(page.getByTestId('v1-sidebar')).toBeVisible({
      timeout: 30_000,
    });
  }

  await expect(page).toHaveURL(/\/tenants\/.+/, { timeout: 30_000 });
}

function sidebar(page: Page) {
  return page.getByTestId('v1-sidebar');
}

function resizeHandle(page: Page) {
  return page.getByTestId('v1-sidebar-resize-handle');
}

// Clicks near the top of the handle: hovering its middle reveals the toggle
// button, which would receive the click instead of the handle.
function clickResizeHandle(page: Page) {
  return resizeHandle(page).click({ position: { x: 2, y: 10 } });
}

async function expectSidebarWidthStyle(page: Page, px: number) {
  await expect(sidebar(page)).toHaveAttribute(
    'style',
    new RegExp(`width: ${px}px`),
  );
}

async function waitForShell(page: Page) {
  await expect(sidebar(page)).toBeVisible({ timeout: 30_000 });
}

test.describe('v1 sidebar: resize + collapse', () => {
  test('navbar: sidebar toggle button is only visible on mobile', async ({
    page,
  }) => {
    await visitAuthed(page, DESKTOP);
    await expect(
      page.locator('button[aria-label="Toggle sidebar"]'),
    ).toBeHidden();

    await visitAuthed(page, MOBILE, { requireVisibleSidebar: false });
    await expect(
      page.locator('button[aria-label="Toggle sidebar"]'),
    ).toBeVisible();
  });

  test('desktop: click the resize edge toggles collapsed/expanded', async ({
    page,
  }) => {
    await visitAuthed(page, DESKTOP);
    await expectSidebarWidthStyle(page, DEFAULT_EXPANDED_WIDTH);

    // Click without dragging -> collapse.
    await clickResizeHandle(page);
    await expectSidebarWidthStyle(page, COLLAPSED_WIDTH);

    // Click again -> expand to last width.
    await clickResizeHandle(page);
    await expectSidebarWidthStyle(page, DEFAULT_EXPANDED_WIDTH);
  });

  test('reload: keeps collapsed state', async ({ page }) => {
    await visitAuthed(page, DESKTOP);

    await clickResizeHandle(page);
    await expectSidebarWidthStyle(page, COLLAPSED_WIDTH);

    await page.reload();
    await waitForShell(page);
    await expectSidebarWidthStyle(page, COLLAPSED_WIDTH);

    // Collapsed mode should render icon buttons with aria-labels.
    await expect(page.locator('button[aria-label="Runs"]')).toBeVisible();
    await expect(page.locator('button[aria-label="Events"]')).toBeVisible();
  });

  test('reload: keeps expanded width', async ({ page }) => {
    await visitAuthed(page, DESKTOP);

    await page.evaluate(() => {
      localStorage.setItem('v1SidebarCollapsed', 'false');
      localStorage.setItem('v1SidebarWidthExpanded', '420');
    });

    await page.reload();
    await waitForShell(page);
    await expectSidebarWidthStyle(page, 420);
  });

  // Skipped before the Playwright migration; kept so the drag behaviour has a
  // ready-made test once it is stable.
  test.skip('desktop: dragging the resize edge updates width and snaps to collapsed when dragged narrow', async ({
    page,
  }) => {
    await visitAuthed(page, DESKTOP);
    const box = await resizeHandle(page).boundingBox();
    if (!box) {
      throw new Error('resize handle has no bounding box');
    }
    const y = box.y + 10;

    // Drag left enough to collapse snap (mouseup should snap to 56).
    await page.mouse.move(box.x + box.width / 2, y);
    await page.mouse.down();
    await page.mouse.move(50, y, { steps: 5 });
    await page.mouse.up();
    await expectSidebarWidthStyle(page, COLLAPSED_WIDTH);

    // Drag right enough to expand snap (mouseup should restore expanded).
    const collapsedBox = await resizeHandle(page).boundingBox();
    if (!collapsedBox) {
      throw new Error('resize handle has no bounding box');
    }
    await page.mouse.move(collapsedBox.x + collapsedBox.width / 2, y);
    await page.mouse.down();
    await page.mouse.move(350, y, { steps: 5 });
    await page.mouse.up();
    await expect(sidebar(page)).toHaveAttribute('style', /width: /);
  });

  test('collapsed: icons show tooltips on the right and active state highlights the current route', async ({
    page,
  }) => {
    await visitAuthed(page, DESKTOP);

    await clickResizeHandle(page);
    await expectSidebarWidthStyle(page, COLLAPSED_WIDTH);

    // Radix tooltips open after a delay; the assertion timeout covers it.
    await page.locator('button[aria-label="Runs"]').hover();
    await expect(
      page.locator('[data-side="right"]', { hasText: 'Runs' }).first(),
    ).toBeVisible({ timeout: 10_000 });

    await page.locator('button[aria-label="Events"]').click();
    await expect(page).toHaveURL(/\/events/, { timeout: 30_000 });
    await expect(page.locator('button[aria-label="Events"]')).toHaveClass(
      /bg-slate-200/,
    );
  });

  test('collapsed: hover affordance button appears and toggles collapsed/expanded', async ({
    page,
  }) => {
    await visitAuthed(page, DESKTOP);

    await resizeHandle(page).hover();
    await page.getByTestId('v1-sidebar-resize-toggle').click();
    await expectSidebarWidthStyle(page, COLLAPSED_WIDTH);
  });

  test('collapsed: settings items are always visible without a flyout', async ({
    page,
  }) => {
    await visitAuthed(page, DESKTOP);

    await clickResizeHandle(page);
    await expectSidebarWidthStyle(page, COLLAPSED_WIDTH);

    const settings = page.locator('button[aria-label="Settings"]');
    await settings.scrollIntoViewIfNeeded();
    await expect(settings).toBeVisible();
  });
});
