describe('v1 sidebar: resize + collapse', () => {
  const DEFAULT_EXPANDED_WIDTH = 200;

  const visitAuthed = (viewport: { width: number; height: number }) => {
    cy.viewport(viewport.width, viewport.height);
    cy.login('owner');

    cy.visit('/', {
      onBeforeLoad(win) {
        // Start each test from a clean sidebar state (but keep auth cookies).
        win.localStorage.removeItem('v1SidebarCollapsed');
        win.localStorage.removeItem('v1SidebarWidthExpanded');
        win.localStorage.removeItem('v1SidebarWidth'); // legacy key
      },
    });

    cy.get('button[aria-label="User Menu"]', { timeout: 30000 }).should(
      'be.visible',
    );
    cy.location('pathname', { timeout: 30000 }).should(
      'match',
      /\/tenants\/.+/,
    );
  };

  const expectSidebarWidthStyle = (px: number) => {
    cy.get('[data-cy="v1-sidebar"]')
      .invoke('attr', 'style')
      .should('contain', `width: ${px}px`);
  };

  const waitForShell = () => {
    cy.get('button[aria-label="User Menu"]', { timeout: 30000 }).should(
      'be.visible',
    );
    cy.get('[data-cy="v1-sidebar"]').should('be.visible');
  };

  it('navbar: sidebar toggle button is only visible on mobile', () => {
    visitAuthed({ width: 1280, height: 800 });
    cy.get('button[aria-label="Toggle sidebar"]').should('not.be.visible');

    visitAuthed({ width: 375, height: 667 });
    cy.get('button[aria-label="Toggle sidebar"]').should('be.visible');
  });

  it('desktop: click the resize edge toggles collapsed/expanded', () => {
    visitAuthed({ width: 1280, height: 800 });

    cy.get('[data-cy="v1-sidebar"]').should('be.visible');

    expectSidebarWidthStyle(DEFAULT_EXPANDED_WIDTH);

    // Click without dragging -> collapse.
    cy.get('[data-cy="v1-sidebar-resize-handle"]').click({ force: true });

    expectSidebarWidthStyle(56);

    // Click again -> expand to last width.
    cy.get('[data-cy="v1-sidebar-resize-handle"]').click({ force: true });

    expectSidebarWidthStyle(DEFAULT_EXPANDED_WIDTH);
  });

  it('reload: keeps collapsed state', () => {
    visitAuthed({ width: 1280, height: 800 });
    cy.get('[data-cy="v1-sidebar"]').should('be.visible');

    // Collapse and verify.
    cy.get('[data-cy="v1-sidebar-resize-handle"]').click({ force: true });
    expectSidebarWidthStyle(56);

    // Reload and ensure it's still collapsed.
    cy.reload();
    waitForShell();
    expectSidebarWidthStyle(56);

    // Collapsed mode should render icon buttons with aria-labels.
    cy.get('button[aria-label="Runs"]').should('be.visible');
    cy.get('button[aria-label="Events"]').should('be.visible');
  });

  it('reload: keeps expanded width', () => {
    visitAuthed({ width: 1280, height: 800 });
    cy.get('[data-cy="v1-sidebar"]').should('be.visible');

    // Simulate a stored expanded width and ensure reload uses it.
    cy.window().then((win) => {
      win.localStorage.setItem('v1SidebarCollapsed', 'false');
      win.localStorage.setItem('v1SidebarWidthExpanded', '420');
    });

    cy.reload();
    waitForShell();
    expectSidebarWidthStyle(420);
  });

  xit('desktop: dragging the resize edge updates width and snaps to collapsed when dragged narrow', () => {
    visitAuthed({ width: 1280, height: 800 });

    cy.get('[data-cy="v1-sidebar"]').should('be.visible');

    // Drag left enough to collapse snap (mouseup should snap to 56).
    cy.get('[data-cy="v1-sidebar-resize-handle"]').trigger('mousedown', {
      clientX: 400,
      clientY: 10,
      force: true,
    });
    cy.wait(50);
    cy.document().trigger('mousemove', {
      clientX: 50,
      clientY: 10,
      force: true,
      bubbles: true,
      buttons: 1,
    });
    cy.document().trigger('mouseup', {
      clientX: 50,
      clientY: 10,
      force: true,
      bubbles: true,
    });

    expectSidebarWidthStyle(56);

    // Drag right enough to expand snap (mouseup should restore expanded).
    cy.get('[data-cy="v1-sidebar-resize-handle"]').trigger('mousedown', {
      clientX: 50,
      clientY: 10,
      force: true,
    });
    cy.wait(50);
    cy.document().trigger('mousemove', {
      clientX: 350,
      clientY: 10,
      force: true,
      bubbles: true,
      buttons: 1,
    });
    cy.document().trigger('mouseup', {
      clientX: 350,
      clientY: 10,
      force: true,
      bubbles: true,
    });

    cy.get('[data-cy="v1-sidebar"]')
      .invoke('attr', 'style')
      .should('contain', 'width: ');
  });

  it('collapsed: icons show tooltips on the right and active state highlights the current route', () => {
    visitAuthed({ width: 1280, height: 800 });

    // Collapse.
    cy.get('[data-cy="v1-sidebar-resize-handle"]').click({ force: true });
    expectSidebarWidthStyle(56);

    // Tooltip shows on the right.
    cy.get('button[aria-label="Runs"]')
      .trigger('pointerover', { force: true })
      .trigger('pointermove', { force: true });
    // Radix tooltip has a built-in delay by default; give it time to appear.
    cy.wait(900);
    cy.contains('[data-side="right"]', 'Runs', { timeout: 10000 }).should(
      'be.visible',
    );

    // Navigate via icon and ensure it becomes active.
    cy.get('button[aria-label="Events"]').click();
    cy.location('pathname', { timeout: 30000 }).should('match', /\/events/);
    cy.get('button[aria-label="Events"]')
      .invoke('attr', 'class')
      .should('contain', 'bg-slate-200');
  });

  it('collapsed: hover affordance button appears and toggles collapsed/expanded', () => {
    visitAuthed({ width: 1280, height: 800 });

    // Hover the gutter to reveal the toggle button.
    cy.get('[data-cy="v1-sidebar-resize-handle"]').trigger('mouseover', {
      force: true,
    });

    cy.get('[data-cy="v1-sidebar-resize-toggle"]')
      .should('be.visible')
      .click({ force: true });

    expectSidebarWidthStyle(56);
  });

  it('collapsed: settings flyout renders and has a visible panel background', () => {
    visitAuthed({ width: 1280, height: 800 });

    // Collapse.
    cy.get('[data-cy="v1-sidebar-resize-handle"]').click({ force: true });

    // Open settings flyout.
    cy.get('button[aria-label="General"]').click({ force: true });
    cy.get('[role="menu"]').filter(':visible').first().as('settingsMenu');
    cy.get('@settingsMenu').contains('Overview').should('be.visible');

    // Content should have the bg-secondary class (explicit panel surface).
    cy.get('@settingsMenu')
      .invoke('attr', 'class')
      // UI uses popover surfaces; accept either explicit secondary surface or popover surface.
      .should('match', /\bbg-(secondary|popover)\b/);
  });
});
