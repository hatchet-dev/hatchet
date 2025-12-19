describe('Error pages', () => {
  describe('authenticated routes', () => {
    beforeEach(() => {
      cy.login('owner');
      cy.visit('/');
    });

    const nonExistentId = '00000000-0000-0000-0000-000000000000';

    function getTenantFromLocation(): Cypress.Chainable<string> {
      return cy
        .location('pathname', { timeout: 30000 })
        .should('match', /\/tenants\/[^/]+/)
        .then((pathname) => {
          const match = pathname.match(/\/tenants\/([^/]+)/);
          expect(
            match,
            `expected tenant in pathname: ${pathname}`,
          ).to.not.equal(null);
          return match![1];
        });
    }

    it('shows tenant forbidden when tenant param is not in memberships', () => {
      cy.visit(`/tenants/missing-tenant/runs`);

      cy.contains('Access denied').should('be.visible');
      cy.contains('Requested Tenant').should('be.visible');
    });

    it('shows organization not found for invalid organization ID', () => {
      cy.visit(`/organizations/${nonExistentId}`);
      cy.contains('Organization not found').should('be.visible');
    });

    it('shows workflow not found for invalid workflow ID', () => {
      getTenantFromLocation().then((tenant) => {
        cy.visit(`/tenants/${tenant}/workflows/${nonExistentId}`);
        cy.contains('Workflow not found').should('be.visible');
      });
    });

    it('shows run not found for invalid run ID', () => {
      getTenantFromLocation().then((tenant) => {
        cy.visit(`/tenants/${tenant}/runs/${nonExistentId}`);
        cy.contains('Run not found').should('be.visible');
      });
    });

    it('shows run not found for malformed run ID (e.g. UUID with extra char)', () => {
      const malformedRunId = '398948ac-18a9-44bc-8c57-b41d399e48cx';
      getTenantFromLocation().then((tenant) => {
        cy.visit(`/tenants/${tenant}/runs/${malformedRunId}`);
        cy.contains('Run not found').should('be.visible');
      });
    });

    it('shows worker not found for invalid worker ID', () => {
      getTenantFromLocation().then((tenant) => {
        cy.visit(`/tenants/${tenant}/workers/${nonExistentId}`);
        cy.contains('Worker not found').should('be.visible');
      });
    });

    it('shows custom 404 for unknown unnested routes', () => {
      cy.visit(`/x`);
      cy.contains('Page not found').should('be.visible');
    });

    it('shows custom 404 for unknown nested routes', () => {
      cy.visit(`/`);
      cy.wait(1000);
      cy.location().then((location) => {
        const path = location.pathname;
        cy.visit(`${path}x`);
        cy.contains('Page not found').should('be.visible');
      });
    });
  });
});
