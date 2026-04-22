describe('auth: logout', () => {
  it('logs out a user', () => {
    // Some environments don't have an "admin" seeded user; "owner" is sufficient to validate logout.
    cy.login('owner');
    cy.visit('/');
    cy.contains('button', 'Logout').filter(':visible').first().click();
    cy.location('pathname').should('include', '/auth/login');
  });
});
