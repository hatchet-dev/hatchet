import { defineConfig } from 'cypress';

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


