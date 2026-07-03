describe('side panel: full height', () => {
  it('scheduled run side panel fills the available vertical space', () => {
    cy.viewport(1280, 800);
    cy.login('owner');

    const workflowId = 'a0000000-0000-0000-0000-000000000001';
    const scheduledRunId = 'b0000000-0000-0000-0000-000000000002';
    const now = '2026-01-01T00:00:00.000Z';

    // Stub a registered workflow so the no-workflows onboarding placeholder
    // does not replace the scheduled runs table.
    cy.intercept('GET', '/api/v1/tenants/*/workflows*', {
      statusCode: 200,
      body: {
        rows: [
          {
            metadata: { id: workflowId, createdAt: now, updatedAt: now },
            name: 'test-workflow',
          },
        ],
        pagination: { num_pages: 1 },
      },
    }).as('workflows');

    // Stub a scheduled run so a row is available to open the side panel.
    cy.intercept('GET', '/api/v1/tenants/*/workflows/scheduled*', {
      statusCode: 200,
      body: {
        rows: [
          {
            metadata: { id: scheduledRunId, createdAt: now, updatedAt: now },
            tenantId: '00000000-0000-0000-0000-000000000000',
            workflowVersionId: workflowId,
            workflowId,
            workflowName: 'test-workflow',
            triggerAt: '2026-12-31T00:00:00.000Z',
            method: 'API',
          },
        ],
        pagination: { num_pages: 1 },
      },
    }).as('scheduledRuns');

    cy.visit('/');
    cy.location('pathname', { timeout: 30000 })
      .should('match', /\/tenants\/.+/)
      .then((pathname) => {
        const tenantBase = pathname.match(/\/tenants\/[^/]+/)?.[0];
        if (!tenantBase) {
          throw new Error(
            `Failed to derive tenant base path from: ${pathname}`,
          );
        }

        cy.visit(`${tenantBase}/scheduled`);
      });

    cy.wait(['@scheduledRuns', '@workflows']);

    // Clicking the run row opens the scheduled run details side panel.
    cy.contains(scheduledRunId).should('be.visible').click();

    cy.get('[data-cy="side-panel"]').should('be.visible');

    // The panel content should fill the available vertical space, extending
    // to the bottom of the side panel.
    cy.get('[data-cy="side-panel"]').then(($panel) => {
      const panelRect = $panel[0].getBoundingClientRect();

      cy.get('[data-cy="side-panel-content"]').then(($content) => {
        const contentRect = $content[0].getBoundingClientRect();
        expect(contentRect.bottom, 'content bottom').to.be.closeTo(
          panelRect.bottom,
          2,
        );
        expect(contentRect.height, 'content height').to.be.greaterThan(500);
      });
    });
  });
});
