/* eslint-disable no-loop-func, @typescript-eslint/no-loop-func */
import { loginSession } from '../../support/flows/auth';

type Viewport = { name: string; width: number; height: number };

const viewports: Viewport[] = [
  { name: 'mobile-shorty', width: 320, height: 200 },
  { name: 'mobile-small', width: 375, height: 667 },
  { name: 'mobile-tall', width: 390, height: 844 },
  { name: 'tablet', width: 768, height: 1024 },
  { name: 'desktop', width: 1280, height: 800 },
];

describe('layout: viewports', () => {
  describe('auth pages', () => {
    beforeEach(() => {
      // These tests must be runnable even if a previous spec logged in and left auth cookies behind.
      // Otherwise `/auth/login` and `/auth/register` can redirect into the app shell and the layout
      // assertions become meaningless (and flaky across spec ordering).
      Cypress.session.clearAllSavedSessions();
      cy.clearAllCookies();
      cy.clearAllLocalStorage();
    });

    for (const vp of viewports) {
      it(`[${vp.name}] login: can reach top and bottom content`, () => {
        cy.viewport(vp.width, vp.height);
        cy.visit('/auth/login');

        cy.location('pathname', { timeout: 30000 }).should('eq', '/auth/login');
        cy.get('[data-cy="auth-title"]', { timeout: 30000 })
          .should('be.visible')
          .and('contain', 'Log in');

        // Ensure the legal text is reachable (scroll container is the auth route wrapper).
        cy.get('[data-cy="auth-legal"]').scrollIntoView().should('be.visible');
        cy.get('[data-cy="auth-title"]').scrollIntoView().should('be.visible');
      });

      it(`[${vp.name}] register: can reach top and bottom content`, () => {
        cy.viewport(vp.width, vp.height);
        cy.visit('/auth/register');

        cy.location('pathname', { timeout: 30000 }).should(
          'eq',
          '/auth/register',
        );
        cy.get('[data-cy="auth-title"]', { timeout: 30000 })
          .should('be.visible')
          .and('contain', 'Create an account');

        cy.get('[data-cy="auth-legal"]').scrollIntoView().should('be.visible');
        cy.get('[data-cy="auth-title"]').scrollIntoView().should('be.visible');
      });
    }
  });

  describe('v1 shell', () => {
    for (const vp of viewports) {
      it(`[${vp.name}] sidebar scrolls and footer stays visible`, () => {
        loginSession('owner');
        cy.viewport(vp.width, vp.height);
        cy.visit('/', {
          onBeforeLoad(win) {
            // Start each test from a clean sidebar state (but keep auth cookies).
            win.localStorage.removeItem('v1SidebarCollapsed');
            win.localStorage.removeItem('v1SidebarWidthExpanded');
            win.localStorage.removeItem('v1SidebarWidth'); // legacy key
          },
        });

        // Wait for the authenticated shell to load (avoids flaking on redirects/hydration).
        cy.get('button[aria-label="User Menu"]', { timeout: 30000 }).should(
          'be.visible',
        );
        cy.location('pathname', { timeout: 30000 }).should(
          'match',
          /\/tenants\/.+/,
        );

        // On narrow viewports the sidebar is closed by default; open it via the hamburger button.
        if (vp.width < 768) {
          cy.get('button[aria-label="Toggle sidebar"]', { timeout: 30000 })
            .should('be.visible')
            .click();
        }

        cy.get('[data-cy="v1-sidebar"]').should('be.visible');
        cy.get('[data-cy="v1-sidebar-footer"]').should('be.visible');

        // Footer stays visible when scrolling the nav list (only if the nav list overflows).
        cy.get('[data-cy="v1-sidebar-scroll"]').then(($el) => {
          const el = $el.get(0);
          const isVertScrollable = el.scrollHeight > el.clientHeight;

          if (!isVertScrollable) {
            cy.log('sidebar nav does not overflow at this viewport');
            return;
          }

          cy.wrap($el).scrollTo('bottom');
          cy.get('[data-cy="v1-sidebar-footer"]').should('be.visible');

          cy.wrap($el).scrollTo('top');
          cy.contains('h2', 'Activity').should('be.visible');
        });

        // Collapsed sidebar should also be scrollable on wide viewports.
        if (vp.width >= 768) {
          // Collapse via the resize edge click (no drag).
          cy.get('[data-cy="v1-sidebar-resize-handle"]').click({ force: true });

          cy.get('[data-cy="v1-sidebar-scroll-collapsed"]').then(($el) => {
            const el = $el.get(0);
            const isVertScrollable = el.scrollHeight > el.clientHeight;

            if (!isVertScrollable) {
              cy.log(
                'collapsed sidebar nav does not overflow at this viewport',
              );
              return;
            }

            cy.wrap($el).scrollTo('bottom');
            cy.wrap($el).invoke('scrollTop').should('be.greaterThan', 0);

            // Help is in the fixed footer and should remain reachable/visible.
            cy.get('button[aria-label="Help Menu"]').should('be.visible');
          });
        }
      });
    }
  });
});
