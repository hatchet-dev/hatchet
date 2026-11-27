import { defineConfig, devices } from '@playwright/test';

const isCI = !!process.env.CI;

// Projects select tests by the tags in e2e/support/test.ts.
export default defineConfig({
  testDir: './e2e',
  forbidOnly: isCI,
  timeout: 120_000,
  expect: { timeout: 15_000 },
  reporter: isCI
    ? [['list'], ['github'], ['html', { open: 'never' }]]
    : [['list'], ['html', { open: 'never' }]],
  use: {
    ...devices['Desktop Chrome'],
    // Local default uses the dev domain (Caddy). CI overrides via PLAYWRIGHT_BASE_URL.
    baseURL:
      process.env.PLAYWRIGHT_BASE_URL || 'https://app.dev.hatchet-tools.com',
    // The local Caddy domain serves a locally issued certificate.
    ignoreHTTPSErrors: true,
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'seeded',
      grep: /@seeded(?=\s|$)/,
      // Specs share one seeded database and mutate the same users (invites,
      // tenant memberships), so they must not run concurrently.
      fullyParallel: false,
      workers: 1,
    },
    {
      name: 'isolated',
      grep: /@isolated(?=\s|$)/,
      fullyParallel: true,
    },
  ],
});
