import * as magicLinkHelpers from '../pages/auth/magic-link/magic-link';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import { runInNewContext } from 'node:vm';
// eslint-disable-next-line import/no-extraneous-dependencies -- Compile the entry point only in this regression test.
import ts from 'typescript';

function compile(relativePath: string) {
  return ts.transpileModule(
    readFileSync(new URL(relativePath, import.meta.url), 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.CommonJS } },
  ).outputText;
}

const entrySource = compile('../main.tsx');
const bootstrapSource = compile('./magic-link-bootstrap.ts');

async function startApp(href: string) {
  const historyState = { navigation: 'preserved' };
  const browser = {
    location: new URL(href),
    history: {
      state: historyState,
      replaceState(state: typeof historyState, _title: string, url: string) {
        browser.location = new URL(url, browser.location);
        browser.history.state = state;
      },
    },
  };
  const bootstrap = { initialMagicLinkToken: null as string | null };
  let telemetryUrl: string | undefined;

  // Execute the real entry point and bootstrap. Observe the URL at the earliest
  // point an app dependency (including the router or Sentry) can evaluate.
  await new Promise<void>((resolve, reject) => {
    runInNewContext(entrySource, {
      exports: {},
      require(id: string) {
        try {
          if (id === './lib/magic-link-bootstrap') {
            runInNewContext(bootstrapSource, {
              window: browser,
              exports: bootstrap,
              require(helperId: string) {
                assert.equal(helperId, '../pages/auth/magic-link/magic-link');
                return magicLinkHelpers;
              },
            });
            return bootstrap;
          }
          assert.equal(id, './app');
          telemetryUrl = browser.location.href;
          resolve();
          return {};
        } catch (error) {
          reject(error);
          return {};
        }
      },
    });
  });

  assert.equal(browser.history.state, historyState);
  return { telemetryUrl, token: bootstrap.initialMagicLinkToken };
}

test('startup removes the credential before app dependencies evaluate', async () => {
  const result = await startApp(
    'https://cloud.example/auth/magic-link?next=%2F#one-time-token',
  );
  assert.equal(
    result.telemetryUrl,
    'https://cloud.example/auth/magic-link?next=%2F',
  );
  assert.equal(result.token, 'one-time-token');
});

test('startup supports base paths and trailing slashes', async () => {
  const result = await startApp(
    'https://cloud.example/hatchet/auth/magic-link/#one-time-token',
  );
  assert.equal(
    result.telemetryUrl,
    'https://cloud.example/hatchet/auth/magic-link/',
  );
  assert.equal(result.token, 'one-time-token');
});

test('startup strips malformed fragments without retaining a token', async () => {
  const result = await startApp(
    'https://cloud.example/auth/magic-link#%E0%A4%A',
  );
  assert.equal(result.telemetryUrl, 'https://cloud.example/auth/magic-link');
  assert.equal(result.token, null);
});

test('startup leaves unrelated fragments intact', async () => {
  const href = 'https://cloud.example/auth/login#section';
  const result = await startApp(href);
  assert.equal(result.telemetryUrl, href);
  assert.equal(result.token, null);
});

test('a reload after scrubbing cannot recover the token', async () => {
  const first = await startApp(
    'https://cloud.example/auth/magic-link#one-time-token',
  );
  const reload = await startApp(first.telemetryUrl!);
  assert.equal(reload.token, null);
});
