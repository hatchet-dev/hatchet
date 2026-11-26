import { defineConfig, devices } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const thisDir = path.dirname(fileURLToPath(import.meta.url));

const seededUsersGeneratedPath = path.resolve(
  thisDir,
  'e2e',
  'support',
  'seeded-users.generated.ts',
);

// The specs import this file, so without it every spec fails with an opaque
// module resolution error. Fail early with the fix instead.
if (!fs.existsSync(seededUsersGeneratedPath)) {
  // eslint-disable-next-line no-console
  console.error(
    [
      '',
      '[playwright] Missing generated seed users file:',
      `  ${seededUsersGeneratedPath}`,
      '',
      'Run the seed task to generate it before running Playwright:',
      '  task seed-e2e',
      '',
      'Alternatively:',
      '  SEED_DEVELOPMENT=true bash ./hack/dev/run-go-with-env.sh run ./cmd/hatchet-admin seed-e2e',
      '',
    ].join('\n'),
  );

  throw new Error('Missing e2e/support/seeded-users.generated.ts');
}

const isCI = !!process.env.CI;

export default defineConfig({
  testDir: './e2e',
  // Specs share one seeded database and mutate the same users (invites,
  // tenant memberships), so they must not run concurrently.
  fullyParallel: false,
  workers: 1,
  // No retries: a flaky spec should fail visibly instead of passing on a rerun.
  retries: 0,
  forbidOnly: isCI,
  timeout: 120_000,
  expect: { timeout: 15_000 },
  reporter: isCI
    ? [['list'], ['github'], ['html', { open: 'never' }]]
    : [['list'], ['html', { open: 'never' }]],
  use: {
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
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        // Matches the viewport the suite was written against; specs that care
        // about layout set their own.
        viewport: { width: 1000, height: 660 },
      },
    },
  ],
});
