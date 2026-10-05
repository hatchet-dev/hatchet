import {
  agentPatterns,
  buildOnboardingPrompt,
  sdkFragments,
} from './prompt-templates';
import assert from 'node:assert/strict';
import { test } from 'node:test';

const base: Parameters<typeof buildOnboardingPrompt>[0] = {
  sdk: 'python',
  patterns: [],
  description: 'process uploaded CSVs and email a summary',
  profileName: 'my-tenant',
};

test('includes the CLI capability preflight', () => {
  const prompt = buildOnboardingPrompt({ ...base });
  assert.match(prompt, /hatchet profile env --help/);
});

test('lists the markdown doc URLs and the SDK reference', () => {
  const prompt = buildOnboardingPrompt({ ...base, sdk: 'typescript' });
  assert.match(prompt, /https:\/\/docs\.hatchet\.run\/llms\.txt/);
  assert.match(
    prompt,
    /https:\/\/docs\.hatchet\.run\/llms\/v1\/quickstart\.md/,
  );
  assert.match(
    prompt,
    new RegExp(sdkFragments.typescript.refDoc.replace(/[.]/g, '\\.')),
  );
});

test('loads credentials from the named profile and never hardcodes the token', () => {
  const prompt = buildOnboardingPrompt({
    ...base,
    sdk: 'go',
    profileName: 'prod-tenant',
  });
  // The two-step form, because eval "$(...)" swallows a failed lookup.
  assert.ok(
    prompt.includes(
      'env_block=$(hatchet profile env --name "prod-tenant") || exit $?',
    ),
  );
  assert.ok(prompt.includes('eval "$env_block"'));
  assert.ok(
    !prompt.includes('eval "$(hatchet profile env --name "prod-tenant")"'),
  );
  assert.match(prompt, /CLI profile named "prod-tenant"/);
  assert.match(prompt, /Do not read, print, or hardcode my token/);
  assert.match(
    prompt,
    /do not use a local or embedded engine for this first run/,
  );
});

test('tells the agent to pass --profile to the CLI, which ignores the env token', () => {
  const prompt = buildOnboardingPrompt({ ...base, profileName: 'prod-tenant' });
  assert.match(prompt, /the CLI does not read HATCHET_CLIENT_TOKEN/);
  assert.ok(prompt.includes('hatchet worker list --profile "prod-tenant"'));
  assert.doesNotMatch(prompt, /It reads the exported HATCHET_CLIENT_TOKEN/);
});

test('always carries the developer description', () => {
  const prompt = buildOnboardingPrompt({ ...base });
  assert.match(
    prompt,
    /Here is what I want to build: process uploaded CSVs and email a summary/,
  );
  assert.match(prompt, /Design the smallest Hatchet workflow in Python/);
});

test('splices in guidance and docs for every selected pattern, in a stable order', () => {
  const prompt = buildOnboardingPrompt({
    ...base,
    // Clicked out of order on purpose.
    patterns: ['cron', 'durable'],
  });
  assert.match(prompt, /Build it with these Hatchet patterns:/);
  assert.ok(prompt.includes(agentPatterns.cron.guidance));
  assert.ok(prompt.includes(agentPatterns.durable.guidance));
  assert.ok(prompt.includes(agentPatterns.cron.doc));
  assert.ok(prompt.includes(agentPatterns.durable.doc));
  assert.ok(
    prompt.indexOf(agentPatterns.durable.guidance) <
      prompt.indexOf(agentPatterns.cron.guidance),
  );
  assert.ok(!prompt.includes(agentPatterns.dag.guidance));
});

test('with no pattern selected, asks the agent to choose the fitting features', () => {
  const prompt = buildOnboardingPrompt({ ...base });
  assert.doesNotMatch(prompt, /Build it with these Hatchet patterns:/);
  assert.match(prompt, /Pick the Hatchet features that fit/);
});

test('treats the install command as an example, not the only way', () => {
  const python = buildOnboardingPrompt({ ...base });
  assert.match(python, /For example: `pip install hatchet-sdk`/);
  assert.match(python, /poetry add hatchet-sdk/);
  assert.match(python, /uv add hatchet-sdk/);
  assert.match(python, /do not introduce a second package manager/);

  const ts = buildOnboardingPrompt({ ...base, sdk: 'typescript' });
  assert.match(ts, /pnpm add @hatchet-dev\/typescript-sdk/);
});

test('asks the agent to fit into an existing project and framework', () => {
  const prompt = buildOnboardingPrompt({ ...base });
  assert.match(prompt, /Fit into my project rather than around it/);
  assert.match(prompt, /FastAPI, Django or Flask/);
});

test('ends with the SDK embeddedLater note', () => {
  const prompt = buildOnboardingPrompt({ ...base, sdk: 'go' });
  assert.ok(prompt.trimEnd().endsWith(sdkFragments.go.embeddedLater));
});
