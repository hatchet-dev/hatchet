import { seededUsers } from '../../support/seeded-users.generated';

describe('Redirect target: deep link restoration', () => {
  it('restores deep link after login', () => {
    cy.request('POST', '/api/v1/users/login', {
      email: seededUsers.owner.email,
      password: seededUsers.owner.password,
    });

    cy.request('GET', '/api/v1/users/memberships/current').then((resp) => {
      const tenantId = resp.body.rows[0].tenant.metadata.id;
      const deepLink = `/tenants/${tenantId}/events`;

      cy.request('POST', '/api/v1/users/logout');

      cy.visit(deepLink);
      cy.location('pathname', { timeout: 15000 }).should('eq', '/auth/login');

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

      cy.location('pathname', { timeout: 30000 }).should('eq', deepLink);
    });
  });

  it('restores deep link after signup and tenant creation', () => {
    const ts = Date.now();
    const userName = `Cypress Redirect ${ts}`;
    const email = `cypress+redirect+${ts}@example.com`;
    const password = `Cypress${ts}pw1`;

    cy.request('POST', '/api/v1/users/login', {
      email: seededUsers.owner.email,
      password: seededUsers.owner.password,
    });

    cy.request('GET', '/api/v1/users/memberships/current').then((resp) => {
      const ownerTenantId = resp.body.rows[0].tenant.metadata.id;
      const deepLink = `/tenants/${ownerTenantId}/events`;

      cy.request('POST', '/api/v1/users/logout');

      cy.visit(deepLink);
      cy.location('pathname', { timeout: 15000 }).should('eq', '/auth/login');

      cy.contains('a', 'Sign up').click();
      cy.location('pathname').should('eq', '/auth/register');

      cy.get('input#name').should('be.visible').clear().type(userName);
      cy.get('input#email').clear().type(email);
      cy.get('input#password').clear().type(password);
      cy.contains('button', 'Create Account').click();

      cy.location('pathname', { timeout: 15000 }).should(
        'eq',
        '/onboarding/create-tenant',
      );

      cy.window().then((win) => {
        expect(win.sessionStorage.getItem('hatchet:redirect_target')).to.eq(
          deepLink,
        );
      });

      const tenantName = `CypressRedirect${String(ts).slice(-6)}`;
      cy.get('input#tenant-name')
        .filter(':visible')
        .first()
        .clear()
        .type(tenantName);
      cy.contains('button', 'Get started').click();

      // The redirect mechanism navigates to the stored path. The new user
      // doesn't own that tenant, so the page shows "Access denied" — but
      // the redirect itself fired correctly.
      cy.location('pathname', { timeout: 30000 }).should('include', deepLink);

      cy.window().then((win) => {
        expect(win.sessionStorage.getItem('hatchet:redirect_target')).to.equal(
          null,
        );
      });
    });
  });
});
