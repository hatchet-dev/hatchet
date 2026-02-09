// eslint-disable-next-line import/no-extraneous-dependencies
import { defineConfig } from 'cypress';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const thisDir = path.dirname(fileURLToPath(import.meta.url));

const seededUsersGeneratedPath = path.resolve(
  thisDir,
  'cypress',
  'support',
  'seeded-users.generated.ts',
);

if (!fs.existsSync(seededUsersGeneratedPath)) {
  // eslint-disable-next-line no-console
  console.error(
    [
      '',
      '[cypress] Missing generated seed users file:',
      `  ${seededUsersGeneratedPath}`,
      '',
      'Run the seed task to generate it before running Cypress:',
      '  task seed-cypress',
      '',
      'Alternatively:',
      '  SEED_DEVELOPMENT=true bash ./hack/dev/run-go-with-env.sh run ./cmd/hatchet-admin seed-cypress',
      '',
    ].join('\n'),
  );

  throw new Error('Missing cypress/support/seeded-users.generated.ts');
}

export default defineConfig({
  e2e: {
    // Local default uses the dev domain (Caddy). CI and other setups can override via CYPRESS_BASE_URL.
    baseUrl:
      process.env.CYPRESS_BASE_URL || 'https://app.dev.hatchet-tools.com',
    specPattern: 'cypress/e2e/**/*.cy.{ts,tsx,js,jsx}',
    supportFile: 'cypress/support/e2e.ts',
    video: true,
    screenshotOnRunFailure: true,
    chromeWebSecurity: true,
    defaultCommandTimeout: 15000,
    requestTimeout: 15000,
    responseTimeout: 30000,
    setupNodeEvents(_on, config) {
      return config;
    },
  },
});
