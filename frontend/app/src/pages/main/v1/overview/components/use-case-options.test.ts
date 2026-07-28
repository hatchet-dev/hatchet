import {
  availableUseCases,
  isLanguageSupported,
  resolveLanguage,
  scaffoldCommand,
  triggerCommand,
} from './use-case-options';
import assert from 'node:assert/strict';
import { test } from 'node:test';

test('scaffoldCommand derives the CLI command from the selection', () => {
  assert.equal(
    scaffoldCommand({ useCase: 'simple', language: 'python' }),
    'hatchet quickstart --use-case simple --language python',
  );
  assert.equal(
    scaffoldCommand({ useCase: 'simple', language: 'typescript' }),
    'hatchet quickstart --use-case simple --language typescript',
  );
  assert.equal(
    scaffoldCommand({ useCase: 'simple', language: 'go' }),
    'hatchet quickstart --use-case simple --language go',
  );
  assert.equal(
    scaffoldCommand({ useCase: 'scheduled', language: 'go' }),
    'hatchet quickstart --use-case scheduled --language go',
  );
});

test('scaffoldCommand resolves an unsupported language to a supported one', () => {
  assert.equal(
    scaffoldCommand({ useCase: 'scheduled', language: 'python' }),
    'hatchet quickstart --use-case scheduled --language go',
  );
});

test('triggerCommand uses the trigger name registered by each template', () => {
  assert.equal(triggerCommand('simple'), 'hatchet trigger simple');
  assert.equal(triggerCommand('scheduled'), 'hatchet trigger manual-run');
});

test('language compatibility matches the published templates', () => {
  assert.equal(isLanguageSupported('simple', 'python'), true);
  assert.equal(isLanguageSupported('simple', 'typescript'), true);
  assert.equal(isLanguageSupported('simple', 'go'), true);
  assert.equal(isLanguageSupported('scheduled', 'go'), true);
  assert.equal(isLanguageSupported('scheduled', 'python'), false);
  assert.equal(isLanguageSupported('scheduled', 'typescript'), false);

  assert.equal(resolveLanguage('simple', 'typescript'), 'typescript');
  assert.equal(resolveLanguage('scheduled', 'typescript'), 'go');
});

test('only the shippable use cases are selectable', () => {
  assert.deepEqual(Object.keys(availableUseCases), ['simple', 'scheduled']);

  // The @ts-expect-error assertions are checked by tsc; the wrapper is
  // never executed.
  const rejectedByTypes = () => {
    // @ts-expect-error roadmap use cases are not selectable
    triggerCommand('pdf');
    // @ts-expect-error roadmap use cases are not selectable
    scaffoldCommand({ useCase: 'claudeAgent', language: 'go' });
  };
  void rejectedByTypes;
});
