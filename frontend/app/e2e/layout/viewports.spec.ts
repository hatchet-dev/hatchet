import { expect, login, test } from '../support/fixtures';
import type { Locator, Page } from '@playwright/test';

type Viewport = { name: string; width: number; height: number };

const viewports: Viewport[] = [
  { name: 'mobile-shorty', width: 320, height: 200 },
  { name: 'mobile-small', width: 375, height: 667 },
  { name: 'mobile-tall', width: 390, height: 844 },
  { name: 'tablet', width: 768, height: 1024 },
  { name: 'desktop', width: 1280, height: 800 },
];

function isVerticallyScrollable(locator: Locator) {
  return locator.evaluate((el) => el.scrollHeight > el.clientHeight);
}

function scrollTo(locator: Locator, position: 'top' | 'bottom') {
  return locator.evaluate((el, pos) => {
    el.scrollTop = pos === 'top' ? 0 : el.scrollHeight;
  }, position);
}

async function expectTopAndBottomReachable(page: Page, title: string) {
  const authTitle = page.getByTestId('auth-title');
  await expect(authTitle).toBeVisible({ timeout: 30_000 });
  await expect(authTitle).toContainText(title);

  // The legal text sits at the bottom of the auth route's scroll container.
  const legal = page.getByTestId('auth-legal');
  await legal.scrollIntoViewIfNeeded();
  await expect(legal).toBeVisible();
  await authTitle.scrollIntoViewIfNeeded();
  await expect(authTitle).toBeVisible();
}

test.describe('layout: viewports', () => {
  test.describe('auth pages', () => {
    for (const vp of viewports) {
      test(`[${vp.name}] login: can reach top and bottom content`, async ({
        page,
      }) => {
        await page.setViewportSize(vp);
        await page.goto('/auth/login');
        await expect(page).toHaveURL((url) => url.pathname === '/auth/login', {
          timeout: 30_000,
        });
        await expectTopAndBottomReachable(page, 'Log in');
      });

      test(`[${vp.name}] register: can reach top and bottom content`, async ({
        page,
      }) => {
        await page.setViewportSize(vp);
        await page.goto('/auth/register');
        await expect(page).toHaveURL(
          (url) => url.pathname === '/auth/register',
          { timeout: 30_000 },
        );
        await expectTopAndBottomReachable(page, 'Create an account');
      });
    }
  });

  test.describe('v1 shell', () => {
    for (const vp of viewports) {
      test(`[${vp.name}] sidebar scrolls and footer stays visible`, async ({
        page,
      }) => {
        await login(page, 'owner');
        await page.setViewportSize(vp);
        // Start each test from a clean sidebar state (but keep auth cookies).
        await page.evaluate(() => {
          localStorage.removeItem('v1SidebarCollapsed');
          localStorage.removeItem('v1SidebarWidthExpanded');
          localStorage.removeItem('v1SidebarWidth'); // legacy key
        });
        await page.goto('/');

        // Wait for the authenticated shell to load (avoids flaking on redirects/hydration).
        await expect(page).toHaveURL(/\/tenants\/.+/, { timeout: 30_000 });
        await expect(page.locator('header')).toBeVisible({ timeout: 30_000 });

        // On narrow viewports the sidebar is closed by default; open it via the hamburger button.
        if (vp.width < 768) {
          await page
            .locator('button[aria-label="Toggle sidebar"]')
            .click({ timeout: 30_000 });
        }

        await expect(page.getByTestId('v1-sidebar')).toBeVisible();
        const footer = page.getByTestId('v1-sidebar-footer');
        await expect(footer).toBeVisible();

        // Footer stays visible when scrolling the nav list (only if the nav list overflows).
        const scroll = page.getByTestId('v1-sidebar-scroll');
        if (await isVerticallyScrollable(scroll)) {
          await scrollTo(scroll, 'bottom');
          await expect(footer).toBeVisible();
          await scrollTo(scroll, 'top');
          await expect(
            page.locator('h2', { hasText: 'Activity' }).first(),
          ).toBeVisible();
        }

        // Collapsed sidebar should also be scrollable on wide viewports.
        if (vp.width >= 768) {
          // Collapse via the resize edge click (no drag). Clicking near the
          // top avoids the toggle button that hovering the middle reveals.
          await page
            .getByTestId('v1-sidebar-resize-handle')
            .click({ position: { x: 2, y: 10 } });

          const collapsedScroll = page.getByTestId(
            'v1-sidebar-scroll-collapsed',
          );
          if (await isVerticallyScrollable(collapsedScroll)) {
            await scrollTo(collapsedScroll, 'bottom');
            await expect
              .poll(() => collapsedScroll.evaluate((el) => el.scrollTop))
              .toBeGreaterThan(0);
            // Help is in the fixed footer and should remain reachable/visible.
            await expect(
              page.locator('button[aria-label="Help Menu"]'),
            ).toBeVisible();
          }
        }
      });
    }
  });
});
