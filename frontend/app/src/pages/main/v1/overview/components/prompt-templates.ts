import { escapeForDoubleQuotes } from './use-case-options';
import { type Sdk } from './use-preferred-sdk';

// Pure prompt-assembly module for the coding-agent onboarding path. No React
// imports, so it is trivially unit-testable.

// Per-SDK fragments spliced into the base wrapper. `name` fills {sdkName};
// `installExample` is ONE example install command and `installAlternatives`
// names the other dependency managers, because the prompt must fit the
// developer's existing toolchain rather than dictate one (a Poetry or uv
// project should not get a stray `pip install`). `frameworks` lists common app
// frameworks the agent should integrate with instead of scaffolding around.
// `connect` is the connect step, `refDoc` the markdown reference URL, and
// `embeddedLater` the closing faster-iteration note.
type SdkFragment = {
  name: string;
  installExample: string;
  installAlternatives: string;
  frameworks: string;
  connect: string;
  refDoc: string;
  embeddedLater: string;
};

export const sdkFragments: Record<Sdk, SdkFragment> = {
  python: {
    name: 'Python',
    installExample: 'pip install hatchet-sdk',
    installAlternatives:
      'with Poetry that is `poetry add hatchet-sdk`, with uv `uv add hatchet-sdk`, with Pipenv `pipenv install hatchet-sdk`',
    frameworks: 'FastAPI, Django or Flask',
    connect:
      'Create the client with `from hatchet_sdk import Hatchet` then `hatchet = Hatchet()`. It reads `HATCHET_CLIENT_TOKEN` from the environment.',
    refDoc: 'https://docs.hatchet.run/llms/reference/python.md',
    embeddedLater:
      'For faster local iteration you can later use `Hatchet.from_embedded()` (a full engine in-process, no server or token). See https://docs.hatchet.run/llms/v1/embedded.md',
  },
  typescript: {
    name: 'TypeScript',
    installExample: 'npm install @hatchet-dev/typescript-sdk',
    installAlternatives:
      'with pnpm that is `pnpm add @hatchet-dev/typescript-sdk`, with Yarn `yarn add @hatchet-dev/typescript-sdk`, with Bun `bun add @hatchet-dev/typescript-sdk`',
    frameworks: 'Next.js, Express or NestJS',
    connect:
      "Create the client with `import { Hatchet } from '@hatchet-dev/typescript-sdk'` then `const hatchet = Hatchet.init()`. It reads `HATCHET_CLIENT_TOKEN` from the environment.",
    refDoc: 'https://docs.hatchet.run/llms/reference/typescript.md',
    embeddedLater:
      'For faster local iteration you can later use `HatchetEmbeddedClient.init()` from `@hatchet-dev/typescript-sdk/v1/embedded` (a full engine in-process, no server or token). See https://docs.hatchet.run/llms/v1/embedded.md',
  },
  go: {
    name: 'Go',
    installExample: 'go get github.com/hatchet-dev/hatchet/sdks/go',
    installAlternatives:
      'inside a Go workspace or a vendored module, follow that setup (for example run `go mod vendor` afterwards)',
    frameworks: 'a net/http, Gin or Echo service',
    connect:
      'Create the client with `hatchet.NewClient()`. It reads `HATCHET_CLIENT_TOKEN` from the environment.',
    refDoc: 'https://docs.hatchet.run/llms/reference/go.md',
    embeddedLater:
      'For faster local iteration you can later use `hatchet.WithEmbedded()` with a blank import of `github.com/hatchet-dev/hatchet-embedded` (a full engine in-process, no server or token). See https://docs.hatchet.run/llms/v1/embedded.md',
  },
};

// The Hatchet patterns offered on the agent path. The developer always
// describes their use case in their own words; patterns are optional hints
// about which Hatchet features it should be built with, and more than one can
// apply (a cron job that kicks off a pipeline). `guidance` is spliced into the
// prompt and `doc` is the markdown page the agent should read for it.
export type AgentPatternKey =
  'durable' | 'dag' | 'background' | 'cron' | 'event';

const LLMS_DOCS = 'https://docs.hatchet.run/llms/v1';

export const agentPatterns: Record<
  AgentPatternKey,
  {
    label: string;
    description: string;
    guidance: string;
    doc: string;
    docPath: string;
  }
> = {
  durable: {
    label: 'Durable execution',
    description:
      'Long-running work that sleeps or waits for events and survives restarts.',
    guidance:
      'Durable execution: use a durable task with durable sleeps and/or wait-for-event steps so the work survives worker restarts and resumes where it left off.',
    doc: `${LLMS_DOCS}/durable-execution.md`,
    docPath: '/v1/durable-execution',
  },
  dag: {
    label: 'DAG / Pipeline',
    description:
      'Multi-step workflows where tasks depend on the output of earlier tasks.',
    guidance:
      'DAG / pipeline: model the work as a workflow of tasks with explicit parent dependencies, passing each parent output to the tasks that depend on it.',
    doc: `${LLMS_DOCS}/directed-acyclic-graphs.md`,
    docPath: '/v1/directed-acyclic-graphs',
  },
  background: {
    label: 'Background task',
    description:
      'Offload work from a request path and run it reliably with retries.',
    guidance:
      'Background task: a task with typed input and output that my application triggers without waiting on it, with sensible retries and a timeout.',
    doc: `${LLMS_DOCS}/running-your-task.md`,
    docPath: '/v1/running-your-task',
  },
  cron: {
    label: 'Cron job',
    description: 'Work that runs on a recurring schedule.',
    guidance:
      'Cron job: run the workflow on a cron schedule (start with every 5 minutes), and show me how to change the schedule and see upcoming runs.',
    doc: `${LLMS_DOCS}/cron-runs.md`,
    docPath: '/v1/cron-runs',
  },
  event: {
    label: 'Webhooks / Event-driven',
    description:
      'Workflows triggered by events you push or by incoming webhooks.',
    guidance:
      'Webhooks / event-driven: trigger the workflow from an event (choose a clear event key), include a small script that pushes a sample event, and if the source is an external service, receive it through a Hatchet webhook.',
    doc: `${LLMS_DOCS}/events.md`,
    docPath: '/v1/events',
  },
};

export const agentPatternOrder: AgentPatternKey[] = [
  'durable',
  'dag',
  'background',
  'cron',
  'event',
];

// Assembles the coding-agent onboarding prompt. `description` is the
// developer's own account of what they want to build (always present on the
// agent path) and `patterns` are the Hatchet patterns they picked, possibly
// none. The result is display + copy only; it is never executed.
export function buildOnboardingPrompt({
  sdk,
  patterns,
  description,
  profileName,
}: {
  sdk: Sdk;
  patterns: AgentPatternKey[];
  description: string;
  // The name of the CLI profile the setup step configured for this tenant.
  // The agent must load credentials from exactly this profile so it connects
  // to the right instance when the developer has several profiles.
  profileName: string;
}): string {
  const fragment = sdkFragments[sdk];
  const profile = escapeForDoubleQuotes(profileName);
  // Keep a stable order regardless of the order the chips were clicked in.
  const selected = agentPatternOrder.filter((key) => patterns.includes(key));

  const patternDocs = selected
    .map((key) => `- ${agentPatterns[key].label}: ${agentPatterns[key].doc}`)
    .join('\n');

  const patternSection =
    selected.length > 0
      ? [
          'Build it with these Hatchet patterns:',
          ...selected.map((key) => `- ${agentPatterns[key].guidance}`),
        ].join('\n')
      : 'Pick the Hatchet features that fit. If it needs a schedule, an event trigger, a multi-step pipeline, or durability, use the matching feature and explain the choice briefly.';

  return `I'm building on Hatchet using the ${fragment.name} SDK. Hatchet is a task orchestration platform for running background tasks, workflows, schedules, and event-driven work.

My setup is already done: the Hatchet CLI is installed and I have a CLI profile named "${profile}" configured with an API token for this tenant. Use that profile by name for everything below (I may have other profiles pointing at other instances, so never rely on the default).

First, confirm my CLI is recent enough for these commands: run \`hatchet profile env --help\`. If that is not a recognized command, my CLI is too old, so upgrade it by re-running the install script before continuing.

Read the docs first. Hatchet's documentation is available as markdown files; fetch the ones you need before writing code:
- Docs index (lists every page and its markdown URL): https://docs.hatchet.run/llms.txt
- ${fragment.name} SDK reference: ${fragment.refDoc}
- Quickstart: https://docs.hatchet.run/llms/v1/quickstart.md
- Running your task: https://docs.hatchet.run/llms/v1/running-your-task.md${patternDocs ? `\n${patternDocs}` : ''}

Fit into my project rather than around it. Look at this directory before writing anything: if there is already an application here (for example ${fragment.frameworks}), follow its layout, configuration and conventions and add Hatchet to it instead of scaffolding a separate project. If the directory is empty, start a minimal project.

Then build the task below, connecting to my LIVE Hatchet instance (do not use a local or embedded engine for this first run):

1. Add the Hatchet SDK with the dependency manager this project already uses. For example: \`${fragment.installExample}\` (${fragment.installAlternatives}). Check for an existing manifest or lockfile first, and do not introduce a second package manager.
2. Connect to my instance. ${fragment.connect} Do not read, print, or hardcode my token. Load my credentials from the "${profile}" CLI profile before running any Hatchet process, using this two-step form: \`env_block=$(hatchet profile env --name "${profile}") || exit $?\` and then \`eval "$env_block"\`. Do not shorten it to a single \`eval "$(...)"\`: command substitution discards the exit code, so a failed profile lookup would silently leave stale or missing credentials. This exports HATCHET_CLIENT_TOKEN and the correct TLS setting for that profile's instance to the SDK, so you never handle the token directly. Run it in the same shell as every worker and script below.
3. Use the \`hatchet\` CLI throughout to check status and inspect runs as you go (run \`hatchet --help\` to discover the available commands). The exported variables are for the SDK only: the CLI does not read HATCHET_CLIENT_TOKEN, and without a profile it opens an interactive picker that fails in a non-interactive shell. So pass \`--profile "${profile}"\` on every \`hatchet\` command that talks to my instance (for example \`hatchet worker list --profile "${profile}"\` or \`hatchet runs list --profile "${profile}"\`).

Here is what I want to build: ${description.trim()}

${patternSection}

Design the smallest Hatchet workflow in ${fragment.name} that accomplishes this.

Finally, start a worker connected to my live instance and trigger a run. Confirm the run completes and instruct the user to navigate back to the dashboard when done.

Once this works, if I want faster local iteration: ${fragment.embeddedLater}`;
}
