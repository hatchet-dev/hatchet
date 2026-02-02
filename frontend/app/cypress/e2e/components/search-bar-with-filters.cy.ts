import type {
  SearchSuggestion,
  AutocompleteResult,
  FilterChip,
} from '../../../src/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';

/**
 * Comprehensive tests for SearchBarWithFilters component
 *
 * Tests basic functionality, keyboard navigation, bug fixes, and edge cases
 */

describe('SearchBarWithFilters', () => {
  // Test fixture: Simple autocomplete implementation
  interface TestSuggestion extends SearchSuggestion<'key' | 'value'> {
    type: 'key' | 'value';
  }

  const testGetAutocomplete = (
    query: string,
  ): AutocompleteResult<TestSuggestion> => {
    const trimmed = query.trimEnd();
    const lastWord = trimmed.split(' ').pop() || '';

    // Handle status: values
    if (lastWord.startsWith('status:')) {
      const partial = lastWord.slice(7).toLowerCase();
      const statuses = ['active', 'inactive', 'pending'];
      const suggestions = statuses
        .filter((s) => s.startsWith(partial))
        .map((s) => ({
          type: 'value' as const,
          label: s,
          value: s,
          description: `Status: ${s}`,
          color: s === 'active' ? 'bg-green-500' : undefined,
        }));
      return { suggestions, mode: 'value' };
    }

    // Handle type: values
    if (lastWord.startsWith('type:')) {
      const partial = lastWord.slice(5).toLowerCase();
      const types = ['user', 'admin', 'guest'];
      const suggestions = types
        .filter((t) => t.startsWith(partial))
        .map((t) => ({
          type: 'value' as const,
          label: t,
          value: t,
          description: `Type: ${t}`,
        }));
      return { suggestions, mode: 'value' };
    }

    // Show filter keys
    const keys: TestSuggestion[] = [
      {
        type: 'key',
        label: 'status',
        value: 'status:',
        description: 'Filter by status',
      },
      {
        type: 'key',
        label: 'type',
        value: 'type:',
        description: 'Filter by type',
      },
    ];

    if (trimmed === '' || query.endsWith(' ')) {
      return { suggestions: keys, mode: 'key' };
    }

    const matchingKeys = keys.filter((k) =>
      k.value.startsWith(lastWord.toLowerCase()),
    );
    if (matchingKeys.length > 0) {
      return { suggestions: matchingKeys, mode: 'key' };
    }

    return { suggestions: [], mode: 'none' };
  };

  const testApplySuggestion = (
    query: string,
    suggestion: TestSuggestion,
  ): string => {
    const trimmed = query.trimEnd();
    const words = trimmed.split(' ');
    const lastWord = words.pop() || '';

    if (suggestion.type === 'value') {
      const prefix = lastWord.slice(0, lastWord.indexOf(':') + 1);
      words.push(prefix + suggestion.value);
    } else {
      const isPartialKey = ['status:', 'type:'].some((key) =>
        key.startsWith(lastWord.toLowerCase()),
      );
      if (lastWord && isPartialKey) {
        words.push(suggestion.value);
      } else {
        words.push(lastWord, suggestion.value);
      }
    }

    return words.filter(Boolean).join(' ');
  };

  const testFilterChips: FilterChip[] = [
    { key: 'status:', label: 'Status', description: 'Filter by status' },
    { key: 'type:', label: 'Type', description: 'Filter by type' },
  ];

  beforeEach(() => {
    // Mount a test page with SearchBarWithFilters
    cy.visit('/test/search-bar-with-filters', {
      onBeforeLoad(win) {
        // Inject test component setup
        (win as any).testConfig = {
          getAutocomplete: testGetAutocomplete,
          applySuggestion: testApplySuggestion,
          filterChips: testFilterChips,
        };
      },
    });
  });

  describe('Basic Input Functionality', () => {
    it('should display placeholder text', () => {
      cy.get('[data-cy="search-bar-input"]').should(
        'have.attr',
        'placeholder',
        'Search...',
      );
    });

    it('should allow typing', () => {
      cy.get('[data-cy="search-bar-input"]').type('test query');
      cy.get('[data-cy="search-bar-input"]').should('have.value', 'test query');
    });

    it('should show clear button when there is input', () => {
      cy.get('[data-cy="search-bar-clear"]').should('not.exist');
      cy.get('[data-cy="search-bar-input"]').type('test');
      cy.get('[data-cy="search-bar-clear"]').should('be.visible');
    });

    it('should clear input when clear button is clicked', () => {
      cy.get('[data-cy="search-bar-input"]').type('test');
      cy.get('[data-cy="search-bar-clear"]').click();
      cy.get('[data-cy="search-bar-input"]').should('have.value', '');
    });
  });

  describe('Autocomplete Dropdown', () => {
    it('should open dropdown on focus', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-suggestions"]').should('be.visible');
    });

    it('should show suggestions while typing', () => {
      cy.get('[data-cy="search-bar-input"]').type('sta');
      cy.get('[data-cy="search-bar-suggestions"]').should('be.visible');
      cy.get('[data-cy="search-bar-suggestion-0"]').should('contain', 'status');
    });

    it('should close dropdown on Escape', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-suggestions"]').should('be.visible');
      cy.get('[data-cy="search-bar-input"]').type('{esc}');
      cy.get('[data-cy="search-bar-suggestions"]').should('not.exist');
    });

    it('should close dropdown after blur delay', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-suggestions"]').should('be.visible');
      cy.get('[data-cy="search-bar-input"]').blur();
      cy.wait(250);
      cy.get('[data-cy="search-bar-suggestions"]').should('not.exist');
    });
  });

  describe('Keyboard Navigation', () => {
    it('should highlight first suggestion on ArrowDown', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-input"]').type('{downarrow}');
      cy.get('[data-cy="search-bar-suggestion-0"]').should(
        'have.class',
        'bg-accent',
      );
    });

    it('should cycle through suggestions with ArrowDown', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-input"]').type('{downarrow}');
      cy.get('[data-cy="search-bar-suggestion-0"]').should(
        'have.class',
        'bg-accent',
      );
      cy.get('[data-cy="search-bar-input"]').type('{downarrow}');
      cy.get('[data-cy="search-bar-suggestion-1"]').should(
        'have.class',
        'bg-accent',
      );
    });

    it('should navigate backwards with ArrowUp', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-input"]').type('{downarrow}{downarrow}');
      cy.get('[data-cy="search-bar-suggestion-1"]').should(
        'have.class',
        'bg-accent',
      );
      cy.get('[data-cy="search-bar-input"]').type('{uparrow}');
      cy.get('[data-cy="search-bar-suggestion-0"]').should(
        'have.class',
        'bg-accent',
      );
    });

    it('should apply selected suggestion on Enter', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-input"]').type('{downarrow}{enter}');
      cy.get('[data-cy="search-bar-input"]').should('have.value', 'status:');
    });

    it('should apply selected suggestion on Tab', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-input"]').type('{downarrow}{tab}');
      cy.get('[data-cy="search-bar-input"]').should('have.value', 'status:');
    });

    it('should submit search on Enter without selection', () => {
      cy.get('[data-cy="search-bar-input"]').type('free text{enter}');
      cy.get('[data-cy="search-bar-input"]').should('have.value', 'free text');
    });
  });

  describe('Mouse Interaction', () => {
    it('should highlight suggestion on hover', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-suggestion-0"]').trigger('mouseenter');
      cy.get('[data-cy="search-bar-suggestion-0"]').should(
        'have.class',
        'bg-accent',
      );
    });

    it('should apply suggestion on click', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-suggestion-0"]').click();
      cy.get('[data-cy="search-bar-input"]').should('have.value', 'status:');
    });
  });

  describe('Filter Chips', () => {
    it('should display filter chips', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-filter-chips"]').should('be.visible');
      cy.get('[data-cy="filter-chip-status"]').should('contain', 'Status');
      cy.get('[data-cy="filter-chip-type"]').should('contain', 'Type');
    });

    it('should append filter key on chip click', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="filter-chip-status"]').click();
      cy.get('[data-cy="search-bar-input"]').should('have.value', 'status:');
    });

    it('should restore focus after chip click', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="filter-chip-status"]').click();
      cy.get('[data-cy="search-bar-input"]').should('have.focus');
    });
  });

  describe('Domain-Specific Autocomplete', () => {
    it('should show status values after "status:"', () => {
      cy.get('[data-cy="search-bar-input"]').type('status:');
      cy.get('[data-cy="search-bar-suggestions"]').should('be.visible');
      cy.get('[data-cy="search-bar-suggestion-0"]').should('contain', 'active');
      cy.get('[data-cy="search-bar-suggestion-1"]').should(
        'contain',
        'inactive',
      );
      cy.get('[data-cy="search-bar-suggestion-2"]').should(
        'contain',
        'pending',
      );
    });

    it('should show type values after "type:"', () => {
      cy.get('[data-cy="search-bar-input"]').type('type:');
      cy.get('[data-cy="search-bar-suggestions"]').should('be.visible');
      cy.get('[data-cy="search-bar-suggestion-0"]').should('contain', 'user');
      cy.get('[data-cy="search-bar-suggestion-1"]').should('contain', 'admin');
      cy.get('[data-cy="search-bar-suggestion-2"]').should('contain', 'guest');
    });

    it('should filter by partial match', () => {
      cy.get('[data-cy="search-bar-input"]').type('status:ac');
      cy.get('[data-cy="search-bar-suggestions"]').should('be.visible');
      cy.get('[data-cy="search-bar-suggestion-0"]').should('contain', 'active');
      cy.get('[data-cy="search-bar-suggestion-1"]').should('not.exist');
    });

    it('should display color indicators', () => {
      cy.get('[data-cy="search-bar-input"]').type('status:');
      cy.get('[data-cy="search-bar-suggestion-0"]')
        .find('.bg-green-500')
        .should('exist');
    });

    it('should close dropdown after selecting a value', () => {
      cy.get('[data-cy="search-bar-input"]').type('status:');
      cy.get('[data-cy="search-bar-suggestion-0"]').click();
      cy.wait(100);
      cy.get('[data-cy="search-bar-suggestions"]').should('not.exist');
    });

    it('should keep dropdown open after selecting a key', () => {
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-suggestion-0"]').click();
      cy.get('[data-cy="search-bar-input"]').should('have.value', 'status:');
      cy.get('[data-cy="search-bar-suggestions"]').should('be.visible');
    });
  });

  describe('Accessibility', () => {
    it('should be fully keyboard accessible', () => {
      // Navigate and select using only keyboard
      cy.get('[data-cy="search-bar-input"]').focus().type('{downarrow}{enter}');
      cy.get('[data-cy="search-bar-input"]').should('have.value', 'status:');

      // Continue navigation
      cy.get('[data-cy="search-bar-input"]').type('{downarrow}{enter}');
      cy.get('[data-cy="search-bar-input"]').should(
        'have.value',
        'status:active',
      );
    });

    it('should maintain focus on input after selection', () => {
      cy.get('[data-cy="search-bar-input"]').focus().type('{downarrow}{enter}');
      cy.get('[data-cy="search-bar-input"]').should('have.focus');
    });
  });

  describe('Bug Fixes', () => {
    it('should not show stale suggestions after space key', () => {
      // Select a value
      cy.get('[data-cy="search-bar-input"]').type('status:{enter}');
      cy.get('[data-cy="search-bar-input"]').should(
        'have.value',
        'status:active',
      );

      // Press space - should show filter keys, not old values
      cy.get('[data-cy="search-bar-input"]').type(' ');
      cy.get('[data-cy="search-bar-suggestions"]').should('be.visible');
      cy.get('[data-cy="search-bar-suggestion-0"]').should('contain', 'status');
      cy.get('[data-cy="search-bar-suggestion-0"]').should(
        'not.contain',
        'active',
      );
    });

    it('should submit empty search when pressing Enter on empty input', () => {
      // Focus empty input - shows suggestions with first item selected
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-suggestions"]').should('be.visible');

      // Press Enter - should submit empty, not apply first suggestion
      cy.get('[data-cy="search-bar-input"]').type('{enter}');
      cy.get('[data-cy="search-bar-input"]').should('have.value', '');
    });

    it('should not show previous selection when refocusing', () => {
      // Make a selection
      cy.get('[data-cy="search-bar-input"]').type('status:{enter}');

      // Close dropdown
      cy.get('body').click(0, 0);
      cy.wait(250);

      // Refocus - should show fresh suggestions with first item selected
      cy.get('[data-cy="search-bar-input"]').focus();
      cy.get('[data-cy="search-bar-suggestions"]').should('be.visible');
      cy.get('[data-cy="search-bar-suggestion-0"]').should(
        'have.class',
        'bg-accent',
      );
    });

    it('should reset selection when suggestions change', () => {
      // Type to show key suggestions
      cy.get('[data-cy="search-bar-input"]').type('sta');
      cy.get('[data-cy="search-bar-suggestion-0"]').should('contain', 'status');

      // Complete the key
      cy.get('[data-cy="search-bar-input"]').type('tus:');

      // Should show value suggestions with first selected
      cy.get('[data-cy="search-bar-suggestion-0"]').should('contain', 'active');
      cy.get('[data-cy="search-bar-suggestion-0"]').should(
        'have.class',
        'bg-accent',
      );
    });
  });
});
