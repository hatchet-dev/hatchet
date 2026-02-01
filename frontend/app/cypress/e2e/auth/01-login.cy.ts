import { seededUsers } from '../../support/seeded-users.generated';

describe('auth: login', () => {
  it('should login a user with username and password', () => {
    cy.visit('/');
    cy.get('input#email').type(seededUsers.owner.email);
    cy.get('input#password').type(seededUsers.owner.password);
    cy.get('form')
      .filter(':visible')
      .first()
      .within(() => {
        cy.contains('button', /^Sign In$/)
          .should('be.enabled')
          .click();
      });
    cy.location('pathname', { timeout: 30000 }).should((pathname) => {
      expect(pathname).to.satisfy(
        (p: string) =>
          p.includes('/tenants/') || p.includes('/onboarding/create-tenant'),
      );
    });

    cy.location('pathname').then((pathname) => {
      if (pathname.includes('/onboarding/create-tenant')) {
        const ts = Date.now();
        const tenantName = `CypressLoginTenant${String(ts).slice(-6)}`;
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
    cy.location('pathname', { timeout: 30000 }).should('include', '/tenants/');
    cy.get('button[aria-label="User Menu"]').filter(':visible').first().click();
    // `data-cy="user-name"` exists in both the trigger and the dropdown content; scope to the open menu.
    cy.get('[role="menu"]')
      .filter(':visible')
      .first()
      .within(() => {
        cy.get('[data-cy="user-name"]')
          .filter(':visible')
          .first()
          .should('have.text', seededUsers.owner.name);
      });
  });
});
