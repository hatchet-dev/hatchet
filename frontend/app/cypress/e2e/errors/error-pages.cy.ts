describe('Error pages', () => {
  describe('authenticated routes', () => {
    beforeEach(() => {
      cy.login('owner');
      cy.visit('/');
    });
    it('shows tenant forbidden when tenant param is not in memberships', () => {
      cy.visit(`/tenants/missing-tenant/runs`);

      cy.contains('Access denied').should('be.visible');
      cy.contains('Requested Tenant').should('be.visible');
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
