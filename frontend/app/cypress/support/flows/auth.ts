import { seededUsers } from '../seeded-users.generated';

/**
 * Logs in and caches the authenticated browser state via `cy.session()`.
 *
 * Uses a programmatic login request for speed/stability, but avoids repeating it for every test.
 * Validation is done by calling `/api/v1/users/current`.
 */
export function loginSession(
  user: keyof typeof seededUsers,
): Cypress.Chainable<null> {
  const { email, password } = seededUsers[user];

  return cy.session(
    ['user-session', email],
    () => {
      // Cypress will retain any auth cookies set by the server for subsequent `cy.visit()` calls.
      cy.request('POST', '/api/v1/users/login', { email, password })
        .its('status')
        .should('eq', 200);

      // Let the SPA hydrate using the authenticated session.
      cy.visit('/');

      // Authenticated root redirect should land you on a tenant route if memberships exist.
      cy.location('pathname', { timeout: 30000 }).should(
        'match',
        /\/tenants\/.+/,
      );
    },
    {
      cacheAcrossSpecs: true,
      validate: () => {
        cy.request({
          method: 'GET',
          url: '/api/v1/users/current',
          failOnStatusCode: false,
        })
          .its('status')
          .should('eq', 200);
      },
    },
  );
}
