describe('Onboarding: create tenant', () => {
  it('logs out a user', () => {
    cy.login('admin');
    cy.visit('/');
  });
});
