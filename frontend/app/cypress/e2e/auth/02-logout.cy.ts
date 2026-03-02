describe('auth: logout', () => {
  it('logs out a user', () => {
    // Some environments don't have an "admin" seeded user; "owner" is sufficient to validate logout.
    cy.login('owner');
    cy.visit('/');
    cy.get('button[aria-label="User Menu"]')
      .filter(':visible')
      .should('be.visible')
      .first()
      .click();
    // Menu item includes a keyboard shortcut, so match by substring.
    cy.contains('[role="menuitem"]', 'Log out').filter(':visible').click();
    cy.location('pathname').should('include', '/auth/login');
  });
});
