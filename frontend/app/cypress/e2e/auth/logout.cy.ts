describe('auth: logout', () => {
  it('logs out a user', () => {
    cy.login('admin');
    cy.visit('/');
    cy.get('button[aria-label="User Menu"]').should('be.visible').click();
    cy.contains('[role="menuitem"]', 'Log out').click();
    cy.location('pathname').should('include', '/auth/login');
  });
});
