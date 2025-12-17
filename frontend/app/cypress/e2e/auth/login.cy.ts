import { seededUsers } from 'cypress/support/seeded-users.generated';

describe('auth: login', () => {
  it('should login a user with username and password', () => {
    cy.visit('/');
    cy.get('input#email').type(seededUsers.owner.email);
    cy.get('input#password').type(seededUsers.owner.password);
    cy.contains('button', 'Sign In').click();
    cy.location('pathname').should('include', '/tenants/');
    cy.get('button[aria-label="User Menu"]').click();
    cy.get('[data-cy="user-name"]').should('have.text', seededUsers.owner.name);
  });
});
