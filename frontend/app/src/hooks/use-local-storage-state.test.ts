// Renders the real hook under jsdom with React's own act, so no test
// framework beyond node:test is involved. The globals must exist before
// react-dom loads, so every import below is dynamic.
import assert from 'node:assert/strict';
import { test } from 'node:test';

// eslint-disable-next-line import/no-extraneous-dependencies -- test-only DOM; never bundled
const { JSDOM } = await import('jsdom');

const dom = new JSDOM('<!doctype html><html><body></body></html>', {
  url: 'http://localhost/',
});

// Newer Node versions expose some of these (navigator) as getter-only
// globals that assignment cannot replace, so they are installed with
// defineProperty. The event classes must be jsdom's, since Node's own are
// not accepted by jsdom's EventTarget.
for (const [name, value] of Object.entries({
  window: dom.window,
  document: dom.window.document,
  navigator: dom.window.navigator,
  localStorage: dom.window.localStorage,
  Event: dom.window.Event,
  CustomEvent: dom.window.CustomEvent,
  StorageEvent: dom.window.StorageEvent,
  IS_REACT_ACT_ENVIRONMENT: true,
})) {
  Object.defineProperty(globalThis, name, { value, configurable: true });
}

const React = (await import('react')).default;
const { act } = await import('react');
const { createRoot } = await import('react-dom/client');
const { useLocalStorageState } = await import('./use-local-storage-state');

const KEY_A = 'hatchet:onboarding:tenant-a';
const KEY_B = 'hatchet:onboarding:tenant-b';

type Marker = { marker: string } | null;

let latest: {
  value: Marker;
  setValue: (value: Marker) => void;
};
const committedValues: Marker[] = [];

function Probe({ storageKey }: { storageKey: string }) {
  const [value, setValue] = useLocalStorageState<Marker>(storageKey, null);
  latest = { value, setValue };
  React.useEffect(() => {
    committedValues.push(value);
  });
  return null;
}

test('the hook follows its key across tenants without leaking state', () => {
  window.localStorage.setItem(KEY_A, JSON.stringify({ marker: 'a' }));
  window.localStorage.setItem(KEY_B, JSON.stringify({ marker: 'b' }));

  const container = document.createElement('div');
  const root = createRoot(container);

  act(() => {
    root.render(React.createElement(Probe, { storageKey: KEY_A }));
  });
  assert.deepEqual(latest.value, { marker: 'a' });

  // committedValues proves no frame of tenant A's state commits under
  // tenant B's key.
  committedValues.length = 0;
  act(() => {
    root.render(React.createElement(Probe, { storageKey: KEY_B }));
  });
  assert.deepEqual(latest.value, { marker: 'b' });
  assert.deepEqual(committedValues, [{ marker: 'b' }]);

  act(() => {
    latest.setValue({ marker: 'b2' });
  });
  assert.deepEqual(latest.value, { marker: 'b2' });
  assert.deepEqual(JSON.parse(window.localStorage.getItem(KEY_B) ?? ''), {
    marker: 'b2',
  });
  assert.deepEqual(JSON.parse(window.localStorage.getItem(KEY_A) ?? ''), {
    marker: 'a',
  });

  act(() => {
    root.render(React.createElement(Probe, { storageKey: KEY_A }));
  });
  assert.deepEqual(latest.value, { marker: 'a' });

  act(() => {
    window.dispatchEvent(
      new dom.window.StorageEvent('storage', {
        key: KEY_B,
        newValue: JSON.stringify({ marker: 'b3' }),
      }),
    );
  });
  assert.deepEqual(latest.value, { marker: 'a' });

  act(() => {
    window.dispatchEvent(
      new dom.window.StorageEvent('storage', {
        key: KEY_A,
        newValue: JSON.stringify({ marker: 'a2' }),
      }),
    );
  });
  assert.deepEqual(latest.value, { marker: 'a2' });

  // hatchet:local-storage is the hook's same-tab change event, dispatched
  // by other hook instances in this document.
  act(() => {
    window.dispatchEvent(
      new dom.window.CustomEvent('hatchet:local-storage', {
        detail: { key: KEY_B, value: { marker: 'b4' } },
      }),
    );
  });
  assert.deepEqual(latest.value, { marker: 'a2' });

  act(() => {
    window.dispatchEvent(
      new dom.window.CustomEvent('hatchet:local-storage', {
        detail: { key: KEY_A, value: { marker: 'a3' } },
      }),
    );
  });
  assert.deepEqual(latest.value, { marker: 'a3' });

  act(() => {
    root.unmount();
  });
});
