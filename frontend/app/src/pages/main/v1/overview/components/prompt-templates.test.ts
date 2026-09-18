import { buildOnboardingPrompt, sdkFragments } from './prompt-templates';
import assert from 'node:assert/strict';
import { test } from 'node:test';

test('includes the CLI capability preflight without an MCP check', () => {
  const prompt = buildOnboardingPrompt({
    sdk: 'python',
    useCaseKey: 'simple',
    profileName: 'my-tenant',
  });
  assert.match(prompt, /hatchet profile env --help/);
  assert.doesNotMatch(prompt, /hatchet mcp --help/);
  assert.doesNotMatch(prompt, /MCP/);
});

test('lists the markdown doc URLs and the SDK reference', () => {
  const prompt = buildOnboardingPrompt({
    sdk: 'typescript',
    useCaseKey: 'scheduled',
    profileName: 'my-tenant',
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

test('loads credentials from the named profile and never hardcodes the token', () => {
  const prompt = buildOnboardingPrompt({
    sdk: 'go',
    useCaseKey: 'simple',
    profileName: 'prod-tenant',
  });
  assert.match(
    prompt,
    /eval "\$\(hatchet profile env --name "prod-tenant"\)"/,
  );
  assert.match(prompt, /Do not read, print, or hardcode my token/);
  assert.match(
    prompt,
    /do not use a local or embedded engine for this first run/,
  );
});

test('names the profile so the agent targets the right instance', () => {
  const prompt = buildOnboardingPrompt({
    sdk: 'python',
    useCaseKey: 'simple',
    profileName: 'staging-eu',
  });
  assert.match(prompt, /CLI profile named "staging-eu"/);
});

test('splices in the selected use-case block', () => {
  const prompt = buildOnboardingPrompt({
    sdk: 'python',
    useCaseKey: 'scheduled',
    profileName: 'my-tenant',
  });
  assert.match(prompt, /runs on a cron schedule/);
});

test('wraps freeform text for the custom use case with the SDK name', () => {
  const prompt = buildOnboardingPrompt({
    sdk: 'go',
    useCaseKey: 'custom',
    freeform: 'process uploaded CSVs and email a summary',
    profileName: 'my-tenant',
  });
  assert.match(prompt, /Here is what I want to build: process uploaded CSVs/);
  assert.match(prompt, /Design the smallest Hatchet workflow in Go/);
});

test('ends with the SDK embeddedLater note', () => {
  const prompt = buildOnboardingPrompt({
    sdk: 'ruby',
    useCaseKey: 'simple',
    profileName: 'my-tenant',
  });
  assert.ok(prompt.trimEnd().endsWith(sdkFragments.ruby.embeddedLater));
});
