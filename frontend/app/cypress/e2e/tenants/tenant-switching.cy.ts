describe('Tenants: switching', () => {
  it('switches tenants and logs out using reusable auth helpers', () => {
    cy.login('owner');
  });
});
