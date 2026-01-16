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

      // Wait for auth redirects to settle; in fast environments the first read can still be "/".
      cy.location('pathname', { timeout: 30000 }).should((pathname) => {
        expect(
          pathname,
          `expected redirect to land on tenant shell or onboarding, got ${pathname}`,
        ).to.satisfy(
          (p: string) =>
            p.includes('/tenants/') || p.includes('/onboarding/create-tenant'),
        );
      });

      cy.location('pathname').then((pathname) => {
        // If the user has no tenant memberships, the app intentionally redirects to onboarding.
        // Complete create-tenant via UI (this triggers the app's own refetch + navigation).
        if (pathname.includes('/onboarding/create-tenant')) {
          const ts = Date.now();
          const tenantName = `CypressSeedTenant${String(ts).slice(-6)}`;

          cy.intercept('POST', '/api/v1/tenants').as('createTenant');
          cy.get('input#name')
            .filter(':visible')
            .first()
            .clear()
            .type(tenantName);
          cy.contains('button', 'Create Tenant').click();
          cy.wait('@createTenant').its('response.statusCode').should('eq', 200);
        }
      });

      cy.location('pathname', { timeout: 30000 }).should(
        'include',
        '/tenants/',
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
