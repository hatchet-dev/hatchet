import { expect } from './test';
import type { Page } from '@playwright/test';

type StoryEvent = { type: string; payload: unknown };

type ActionPayload = { data: { name: string; args: unknown[] } };

declare global {
  interface Window {
    __storyEvents?: StoryEvent[];
  }
}

// A story always ends with one of these.
const END_EVENTS = [
  'storyFinished',
  'storyErrored',
  'storyThrewException',
  'storyMissing',
];

// Outside Storybook's own test runner, a failing play function still ends with
// a successful `storyFinished`, so failures are read from these as well.
const FAILURE_EVENTS = [
  'storyErrored',
  'storyThrewException',
  'storyMissing',
  'playFunctionThrewException',
  'unhandledErrorsWhilePlaying',
];

const ACTION_EVENT = 'storybook/actions/action-event';

// Storybook assigns its preview channel to this global once its runtime boots,
// so trapping the assignment subscribes before the story renders or plays and
// no event is missed.
function recordStoryEvents(eventTypes: string[]) {
  const events: StoryEvent[] = [];
  window.__storyEvents = events;

  type Channel = { on: (type: string, cb: (payload: unknown) => void) => void };
  let channel: Channel | undefined;
  Object.defineProperty(window, '__STORYBOOK_ADDONS_CHANNEL__', {
    configurable: true,
    get: () => channel,
    set(next: Channel) {
      if (next === channel) {
        return;
      }
      channel = next;
      eventTypes.forEach((type) =>
        next.on(type, (payload) => events.push({ type, payload })),
      );
    },
  });
}

function storyEvents(page: Page) {
  return page.evaluate(() => window.__storyEvents ?? []);
}

// Opens a story on its own, without the Storybook UI, and waits until it has
// rendered and its play function, if any, has passed.
export async function gotoStory(page: Page, storyId: string) {
  await page.addInitScript(recordStoryEvents, [
    ...new Set([...END_EVENTS, ...FAILURE_EVENTS, ACTION_EVENT]),
  ]);
  await page.goto(
    `/iframe.html?id=${encodeURIComponent(storyId)}&viewMode=story`,
  );

  await expect
    .poll(
      async () =>
        (await storyEvents(page)).some((event) =>
          END_EVENTS.includes(event.type),
        ),
      { message: `story ${storyId} did not finish` },
    )
    .toBe(true);

  const failures = (await storyEvents(page)).filter(
    (event) =>
      FAILURE_EVENTS.includes(event.type) ||
      (event.type === 'storyFinished' &&
        (event.payload as { status: string }).status !== 'success'),
  );
  expect(
    failures.map((event) => `${event.type}: ${JSON.stringify(event.payload)}`),
    `story ${storyId} failed`,
  ).toEqual([]);
}

// The argument list of every call to an `fn()` arg, which Storybook reports
// as an action named after the arg. Objects arrive serialized, so compare
// fields rather than identity.
export async function storyActionCalls(page: Page, argName: string) {
  return (await storyEvents(page))
    .filter((event) => event.type === ACTION_EVENT)
    .map((event) => (event.payload as ActionPayload).data)
    .filter((data) => data.name === argName)
    .map((data) => data.args);
}
