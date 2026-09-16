import { buildOnboardingPrompt, sdkFragments } from './prompt-templates';
import assert from 'node:assert/strict';
import { test } from 'node:test';

test('includes the CLI capability preflight', () => {
  const prompt = buildOnboardingPrompt({ sdk: 'python', useCaseKey: 'simple' });
  assert.match(prompt, /hatchet profile env --help/);
  assert.match(prompt, /hatchet mcp --help/);
});

test('lists the markdown doc URLs and the SDK reference', () => {
  const prompt = buildOnboardingPrompt({
    sdk: 'typescript',
    useCaseKey: 'scheduled',
  });
  assert.match(prompt, /https:\/\/docs\.hatchet\.run\/llms\.txt/);
  assert.match(
    prompt,
    /https:\/\/docs\.hatchet\.run\/llms\/v1\/quickstart\.md/,
  );
  assert.match(
    prompt,
    /https:\/\/docs\.hatchet\.run\/llms\/v1\/running-your-task\.md/,
  );
  assert.match(
    prompt,
    new RegExp(sdkFragments.typescript.refDoc.replace(/[.]/g, '\\.')),
  );
});

test('loads credentials via eval hatchet profile env and never hardcodes the token', () => {
  const prompt = buildOnboardingPrompt({ sdk: 'go', useCaseKey: 'simple' });
  assert.match(prompt, /eval "\$\(hatchet profile env\)"/);
  assert.match(prompt, /Do not read, print, or hardcode my token/);
  assert.match(
    prompt,
    /do not use a local or embedded engine for this first run/,
  );
});

test('splices in the selected use-case block', () => {
  const prompt = buildOnboardingPrompt({
    sdk: 'python',
    useCaseKey: 'scheduled',
  });
  assert.match(prompt, /runs on a cron schedule/);
});

test('wraps freeform text for the custom use case with the SDK name', () => {
  const prompt = buildOnboardingPrompt({
    sdk: 'go',
    useCaseKey: 'custom',
    freeform: 'process uploaded CSVs and email a summary',
  });
  assert.match(prompt, /Here is what I want to build: process uploaded CSVs/);
  assert.match(prompt, /Design the smallest Hatchet workflow in Go/);
});

test('ends with the SDK embeddedLater note', () => {
  const prompt = buildOnboardingPrompt({ sdk: 'ruby', useCaseKey: 'simple' });
  assert.ok(prompt.trimEnd().endsWith(sdkFragments.ruby.embeddedLater));
});
