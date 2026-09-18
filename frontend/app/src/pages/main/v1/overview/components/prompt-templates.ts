import {
  escapeForDoubleQuotes,
  type AvailableUseCaseKey,
} from './use-case-options';
import { type Sdk } from './use-preferred-sdk';

// Pure prompt-assembly module for the coding-agent onboarding path. No React
// imports, so it is trivially unit-testable. All strings are copied inline
// from docs/plans/onboarding-copy.mdx (sections 5 and 6); keep them in sync
// with that file by hand.

// Per-SDK fragments spliced into the base wrapper. Mirrors
// [copy: sdk.fragments.*]. `name` fills {sdkName}, `install` the install
// step, `connect` the connect step, `refDoc` the markdown reference URL, and
// `embeddedLater` the closing faster-iteration note.
type SdkFragment = {
  name: string;
  install: string;
  connect: string;
  refDoc: string;
  embeddedLater: string;
};

export const sdkFragments: Record<Sdk, SdkFragment> = {
  python: {
    name: 'Python',
    install: 'pip install hatchet-sdk',
    connect:
      'Create the client with `from hatchet_sdk import Hatchet` then `hatchet = Hatchet()`. It reads `HATCHET_CLIENT_TOKEN` from the environment.',
    refDoc: 'https://docs.hatchet.run/llms/reference/python.md',
    embeddedLater:
      'For faster local iteration you can later use `Hatchet.from_embedded()` (a full engine in-process, no server or token). See https://docs.hatchet.run/llms/v1/embedded.md',
  },
  typescript: {
    name: 'TypeScript',
    install: 'npm install @hatchet-dev/typescript-sdk',
    connect:
      "Create the client with `import { Hatchet } from '@hatchet-dev/typescript-sdk'` then `const hatchet = Hatchet.init()`. It reads `HATCHET_CLIENT_TOKEN` from the environment.",
    refDoc: 'https://docs.hatchet.run/llms/reference/typescript.md',
    embeddedLater:
      'For faster local iteration you can later use `HatchetEmbeddedClient.init()` from `@hatchet-dev/typescript-sdk/v1/embedded` (a full engine in-process, no server or token). See https://docs.hatchet.run/llms/v1/embedded.md',
  },
  go: {
    name: 'Go',
    install: 'go get github.com/hatchet-dev/hatchet/sdks/go',
    connect:
      'Create the client with `hatchet.NewClient()`. It reads `HATCHET_CLIENT_TOKEN` from the environment.',
    refDoc: 'https://docs.hatchet.run/llms/reference/go.md',
    embeddedLater:
      'For faster local iteration you can later use `hatchet.WithEmbedded()` with a blank import of `github.com/hatchet-dev/hatchet-embedded` (a full engine in-process, no server or token). See https://docs.hatchet.run/llms/v1/embedded.md',
  },
  ruby: {
    name: 'Ruby (early access)',
    install: 'gem install hatchet-sdk',
    connect:
      'Create the client per the Ruby SDK reference. It reads `HATCHET_CLIENT_TOKEN` from the environment.',
    refDoc: 'https://docs.hatchet.run/llms/reference/ruby.md',
    embeddedLater:
      'Embedded mode is not yet available for Ruby; keep running against your live instance.',
  },
};

// Every use case offered on the agent (prompt-only) path. A superset of the
// scaffoldable AvailableUseCaseKey: the extra keys (fanout, event, durable)
// only drive prompt generation, not CLI scaffolding, so they live on the agent
// path but never reach the manual command builders.
export type AgentUseCaseKey =
  AvailableUseCaseKey | 'fanout' | 'event' | 'durable';

// The use-case blocks spliced in at {useCaseBlock}. Mirrors
// [copy: prompt.usecase.*]. One map keyed by every agent use case; the manual
// path only ever passes a scaffoldable key. `custom` is handled separately
// because it interpolates the freeform text and the SDK name.
export const useCaseBlocks: Record<AgentUseCaseKey, string> = {
  simple:
    'Scaffold a single Hatchet task that takes a typed input and returns a typed output. Give it a clear name, validate the input, and log a line when it runs. Register it on a worker.',
  scheduled:
    'Scaffold a Hatchet workflow that runs on a cron schedule (start with every 5 minutes). The workflow should do one small unit of work and log its result. Show me how to change the schedule and how to see upcoming runs.',
  fanout:
    'Scaffold a Hatchet fan-out workflow: a parent task that splits its input into N items and spawns a child task per item to run in parallel, then a step that aggregates the child results into one output. Make N configurable from the input.',
  event:
    'Scaffold an event-driven Hatchet workflow: a workflow that is triggered by an event I push (choose a clear event key), plus a small script that pushes a sample event. Show me the run that results and its events.',
  durable:
    'Scaffold a durable Hatchet task that does long-running work: use a durable sleep and a wait-for-event step so it survives worker restarts. Show me that the run resumes correctly if the worker is restarted mid-run.',
};

function customUseCaseBlock(sdkName: string, freeform: string): string {
  const text = freeform.trim();
  return [
    `Here is what I want to build: ${text}`,
    '',
    `Design the smallest Hatchet workflow in ${sdkName} that accomplishes this. If it needs a schedule, an event trigger, parallelism, or durability, use the matching Hatchet feature and explain the choice briefly.`,
  ].join('\n');
}

// Assembles the coding-agent onboarding prompt for the selected SDK and use
// case. `useCaseKey` is a selectable use case or the `custom` sentinel, in
// which case `freeform` is wrapped per prompt.usecase.custom. The result is
// display + copy only; it is never executed.
export function buildOnboardingPrompt({
  sdk,
  useCaseKey,
  freeform,
  profileName,
}: {
  sdk: Sdk;
  useCaseKey: AgentUseCaseKey | 'custom';
  freeform?: string;
  // The name of the CLI profile the setup step configured for this tenant.
  // The agent must load credentials from exactly this profile so it connects
  // to the right instance when the developer has several profiles.
  profileName: string;
}): string {
  const fragment = sdkFragments[sdk];
  const profile = escapeForDoubleQuotes(profileName);
  const useCaseBlock =
    useCaseKey === 'custom'
      ? customUseCaseBlock(fragment.name, freeform ?? '')
      : useCaseBlocks[useCaseKey];

  return `I'm building on Hatchet using the ${fragment.name} SDK. Hatchet is a task orchestration platform for running background tasks, workflows, schedules, and event-driven work.

My setup is already done: the Hatchet CLI is installed and I have a CLI profile named "${profile}" configured with an API token for this tenant. Use that profile by name for everything below (I may have other profiles pointing at other instances, so never rely on the default).

First, confirm my CLI is recent enough for these commands: run \`hatchet profile env --help\`. If that is not a recognized command, my CLI is too old, so upgrade it by re-running the install script before continuing.

Read the docs first. Hatchet's documentation is available as markdown files; fetch the ones you need before writing code:
- Docs index (lists every page and its markdown URL): https://docs.hatchet.run/llms.txt
- ${fragment.name} SDK reference: ${fragment.refDoc}
- Quickstart: https://docs.hatchet.run/llms/v1/quickstart.md
- Running your task: https://docs.hatchet.run/llms/v1/running-your-task.md

Then build the task below, connecting to my LIVE Hatchet instance (do not use a local or embedded engine for this first run):

1. Install the SDK: ${fragment.install}
2. Connect to my instance. ${fragment.connect} Do not read, print, or hardcode my token. Load my credentials from the "${profile}" CLI profile with command substitution before running any Hatchet process: \`eval "$(hatchet profile env --name "${profile}")"\`. This exports HATCHET_CLIENT_TOKEN and the correct TLS setting for that profile's instance, so you never handle the token directly. Run it in the same shell as every worker and script below.
3. Use the \`hatchet\` CLI throughout to check status and inspect runs as you go (run \`hatchet --help\` to discover the available commands). It reads the exported HATCHET_CLIENT_TOKEN, so it targets the same instance.

${useCaseBlock}

Finally, start a worker connected to my live instance and trigger a run. Confirm the run completes and instruct the user to navigate back to the dashboard when done.

Once this works, if I want faster local iteration: ${fragment.embeddedLater}`;
}
