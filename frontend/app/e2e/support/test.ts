import { test as base, expect } from '@playwright/test';

export { expect };

// Every test declares what it needs to run, and playwright.config.ts maps each
// tag to its own project:
// - seeded: the full stack (API, engine, database) loaded with `task seed-e2e`.
//   Tests share that database, so the project runs them one at a time.
// - isolated: only a frontend server. Every API call must be mocked with
//   `page.route`, so these tests run in parallel and need no backend.
export const TAGS = {
  seeded: '@seeded',
  isolated: '@isolated',
} as const;

const CATEGORY_TAGS: string[] = Object.values(TAGS);

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

export const test = base.extend<{ testCategory: string }>({
  // Projects select tests by tag, so an untagged test would silently run in
  // none of them.
  testCategory: [
    // Playwright reads fixture dependencies from this destructuring pattern,
    // so it must stay an object pattern even when empty.
    // eslint-disable-next-line no-empty-pattern
    async ({}, use, testInfo) => {
      const categories = testInfo.tags.filter((tag) =>
        CATEGORY_TAGS.includes(tag),
      );
      if (categories.length !== 1) {
        throw new Error(
          `Tag the test with exactly one of ${CATEGORY_TAGS.join(', ')}; found [${categories.join(', ')}]`,
        );
      }

      await use(categories[0]);
    },
    { auto: true },
  ],

  page: async ({ page, testCategory }, use) => {
    // Playwright ignores uncaught page errors by default; fail the test on them
    // so app crashes surface in e2e runs.
    const pageErrors: Error[] = [];
    page.on('pageerror', (err) => {
      if (!isIgnoredPageError(err)) {
        pageErrors.push(err);
      }
    });

    // Registered before the test body runs, so the test's own `page.route`
    // mocks take precedence and only unmocked calls reach this handler.
    const unmockedApiCalls: string[] = [];
    if (testCategory === TAGS.isolated) {
      // Matched on the pathname: a `**/api/**` glob also catches the dev
      // server's source modules under /src/lib/api/.
      await page.route(
        (url) => url.pathname.startsWith('/api/'),
        (route) => {
          const request = route.request();
          unmockedApiCalls.push(
            `${request.method()} ${new URL(request.url()).pathname}`,
          );

          return route.abort('blockedbyclient');
        },
      );
    }

    await use(page);

    expect(unmockedApiCalls, 'API calls without a page.route mock').toEqual([]);
    expect(
      pageErrors.map((err) => `${err.name}: ${err.message}`),
      'uncaught errors in the page',
    ).toEqual([]);
  },
});
