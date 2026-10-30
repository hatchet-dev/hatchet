import { installMethodOptions, type InstallMethod } from './onboarding-options';
import {
  agentPatternOrder,
  agentPatterns,
  buildOnboardingPrompt,
  type AgentPatternKey,
} from './prompt-templates';
import {
  availableUseCases,
  escapeForDoubleQuotes,
  scaffoldCommand,
  triggerCommand,
  workerDevCommand,
  type AvailableUseCaseKey,
} from './use-case-options';
import { type OnboardingProgress } from './use-onboarding-progress';
import {
  focusRing,
  SdkSwitcher,
  segmentedTabsListClass,
  segmentedTabsTriggerClass,
  type Sdk,
} from './use-preferred-sdk';
import { HelpDropdown } from '@/components/v1/nav/help-dropdown';
import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Spinner } from '@/components/v1/ui/loading';
import { RadioGroup, RadioGroupCardItem } from '@/components/v1/ui/radio-group';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { Textarea } from '@/components/v1/ui/textarea';
import { cn } from '@/lib/utils';
import { appRoutes, type OnboardingSearch } from '@/router';
import {
  CheckIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ExternalLinkIcon,
} from '@radix-ui/react-icons';
import { Link } from '@tanstack/react-router';
import { Bot, LifeBuoy, Terminal } from 'lucide-react';
import { useEffect, useMemo, useState, type ReactNode } from 'react';

// Brand-tinted selected state for the use-case cards, layered over the
// RadioGroupCardItem base so selection reads as the app's brand blue instead
// of muted grey and unselected cards gain a brand hover affordance.
const brandCardClass =
  'data-[state=unchecked]:hover:border-brand/50 data-[state=checked]:border-brand data-[state=checked]:hover:border-brand data-[state=checked]:bg-brand/10';

// Path and step are the tenant onboarding route's search params, so their
// types come from that schema rather than being declared twice.
export type SetupPath = NonNullable<OnboardingSearch['path']>;
export type StepKey = NonNullable<OnboardingSearch['step']>;

const stepRailLabels: Record<StepKey, string> = {
  path: 'Choose path',
  usecase: 'Use case & SDK',
  setup: 'Set up CLI',
  runagent: 'Run agent',
  runtask: 'Run task',
  finish: 'Finish',
};

// Docs relevant to the finish step, chosen from the user's path, use case, and
// SDK. All hrefs are real pages under docs.hatchet.run. Rendered as a short
// list of external links so the user knows where to go next.
const DOCS_BASE = 'https://docs.hatchet.run';

const sdkReferenceDocs: Record<Sdk, { label: string; href: string }> = {
  python: {
    label: 'Python SDK reference',
    href: `${DOCS_BASE}/reference/python`,
  },
  typescript: {
    label: 'TypeScript SDK reference',
    href: `${DOCS_BASE}/reference/typescript`,
  },
  go: { label: 'Go SDK reference', href: `${DOCS_BASE}/reference/go` },
};

// Manual-path templates map to one docs page each; the agent path links the
// page for every pattern the developer picked.
const manualUseCaseDocs: Record<
  AvailableUseCaseKey,
  { label: string; href: string }
> = {
  simple: {
    label: 'Running your task',
    href: `${DOCS_BASE}/v1/running-your-task`,
  },
  scheduled: {
    label: 'Scheduled runs',
    href: `${DOCS_BASE}/v1/scheduled-runs`,
  },
};

function relevantDocs({
  sdk,
  useCase,
  patterns,
  path,
}: {
  sdk: Sdk;
  useCase: AvailableUseCaseKey;
  patterns: AgentPatternKey[];
  path: SetupPath | null;
}): { label: string; href: string }[] {
  const docs: { label: string; href: string }[] = [sdkReferenceDocs[sdk]];
  if (path === 'agent') {
    agentPatternOrder
      .filter((key) => patterns.includes(key))
      .forEach((key) =>
        docs.push({
          label: agentPatterns[key].label,
          href: `${DOCS_BASE}${agentPatterns[key].docPath}`,
        }),
      );
    docs.push({
      label: 'MCP server reference',
      href: `${DOCS_BASE}/reference/cli/mcp`,
    });
  } else {
    docs.push(manualUseCaseDocs[useCase]);
  }
  docs.push({
    label: 'Embedded mode for local iteration',
    href: `${DOCS_BASE}/v1/embedded`,
  });
  return docs;
}

const codeBlockClass = 'bg-muted/20 ring-1 ring-border/50 ring-inset px-1';

// Module-level so its identity is stable: defined inside the stepper it would
// remount (restarting the spinner) on every poll and keystroke.
function StatusRow({
  done,
  waiting,
  ready,
}: {
  done: string;
  waiting: string;
  ready: boolean;
}) {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-border/50 bg-muted/20 p-4">
      {ready ? (
        <>
          <CheckIcon className="size-5 text-foreground" />
          <span className="text-sm font-medium text-foreground">{done}</span>
        </>
      ) : (
        <>
          <Spinner className="size-5" />
          <span className="text-sm text-muted-foreground">{waiting}</span>
        </>
      )}
    </div>
  );
}

// The visible step sequence for a path. The path is chosen first, so before a
// path exists the selector is the only step; picking one reveals the rest.
function stepSequence(path: SetupPath | null): StepKey[] {
  if (path === 'agent') {
    return ['path', 'usecase', 'setup', 'runagent', 'finish'];
  }
  if (path === 'manual') {
    return ['path', 'usecase', 'setup', 'runtask', 'finish'];
  }
  return ['path'];
}

export function OnboardingSteps({
  tenantName,
  tenantId,
  path,
  step,
  onNavigate,
  patterns,
  onPatternsChange,
  description,
  onDescriptionChange,
  sdk,
  onSdkChange,
  useCase,
  onUseCaseChange,
  onConfirmSelection,
  profileToken,
  isGeneratingProfileToken,
  profileTokenError,
  onGenerateProfileToken,
  canGenerateToken,
  hasApiToken,
  authDisabled,
  authDisabledToken,
  progress,
  onFinish,
  onPromptGenerated,
  onStepChangeEvent,
}: {
  tenantName?: string;
  // The current tenant id, used to build links to the first completed run and
  // the worker that executed it.
  tenantId?: string;
  // Navigation is controlled by the URL (the tenant onboarding route's search
  // params) so a refresh, back/forward, or a shared link lands on the same
  // step. `step` may be absent or not valid for `path`; it is normalized below.
  path: SetupPath | null;
  step?: StepKey;
  onNavigate: (next: { path: SetupPath | null; step: StepKey }) => void;
  // Agent path: the Hatchet patterns picked (any number, possibly none) and
  // the developer's own description of what they are building, which is always
  // required. Owned by the parent so they survive a refresh too.
  patterns: AgentPatternKey[];
  onPatternsChange: (next: AgentPatternKey[]) => void;
  description: string;
  onDescriptionChange: (next: string) => void;
  sdk: Sdk;
  // Updates the global SDK preference.
  onSdkChange: (next: Sdk) => void;
  // The persisted manual-path template (one the CLI can scaffold).
  useCase: AvailableUseCaseKey;
  onUseCaseChange: (next: AvailableUseCaseKey) => void;
  // Records selectionConfirmedAt so progress polling can begin. Idempotent.
  onConfirmSelection: () => void;
  profileToken?: string;
  isGeneratingProfileToken: boolean;
  profileTokenError?: string;
  onGenerateProfileToken: () => void;
  canGenerateToken: boolean;
  // Whether the tenant has an API token (checked against the API). Gates the
  // Next button on the Set up CLI step so it survives refreshes.
  hasApiToken: boolean;
  authDisabled?: boolean;
  authDisabledToken?: string;
  progress: OnboardingProgress;
  onFinish: () => void;
  // Fired when the agent-path prompt is generated or regenerated.
  onPromptGenerated: (patterns: AgentPatternKey[], sdk: Sdk) => void;
  // Fired on navigation to a different step (the stepper analog of a tab
  // change).
  onStepChangeEvent?: (label: string) => void;
}) {
  const profileName = tenantName?.trim() || 'local';

  const [installMethod, setInstallMethod] = useState<InstallMethod>(
    installMethodOptions.native.value,
  );

  const sequence = useMemo(() => stepSequence(path), [path]);
  // A step from the URL that does not belong to this path's sequence (a stale
  // link, or the other path's run step) falls back to the first step. A link
  // past the use-case step with no description falls back to it, since every
  // later agent step is built on that description.
  const requestedStep: StepKey =
    step && sequence.includes(step) ? step : sequence[0];
  const needsDescription =
    path === 'agent' &&
    description.trim().length === 0 &&
    sequence.indexOf(requestedStep) > sequence.indexOf('usecase');
  const currentStep: StepKey = needsDescription ? 'usecase' : requestedStep;

  // Landing directly on a later step (refresh or a link) bypasses goTo, which
  // is what normally confirms the selection. Progress polling only starts once
  // the selection is confirmed, so confirm it here too (it is idempotent).
  const pastSelection = currentStep !== 'path' && currentStep !== 'usecase';
  useEffect(() => {
    if (pastSelection) {
      onConfirmSelection();
    }
    // onConfirmSelection is a fresh closure each render; keying on the step
    // alone avoids re-confirming on every parent render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pastSelection]);

  // Confirm the selection once the user advances past the use-case step, so
  // progress polling begins then (never on the first steps). confirmSelection
  // is idempotent, so firing on every later step is safe.
  const goTo = (target: StepKey, targetPath: SetupPath | null = path) => {
    if (target !== 'path' && target !== 'usecase') {
      onConfirmSelection();
    }
    if (target !== currentStep) {
      onStepChangeEvent?.(stepRailLabels[target]);
    }
    onNavigate({ path: targetPath, step: target });
  };

  const choosePath = (next: SetupPath) => {
    // Path and step change together in one navigation, so the URL never holds
    // a step that is invalid for its path.
    goTo('usecase', next);
  };

  const generatedPrompt = useMemo(
    () =>
      buildOnboardingPrompt({
        sdk,
        patterns,
        description,
        profileName,
      }),
    [sdk, patterns, description, profileName],
  );

  // Optional MCP-install section. Lets the developer wire the Hatchet MCP into
  // their coding agent before they run the prompt, so the agent can trigger
  // runs and inspect results as it builds. Rendered at the top of the Run agent
  // step; skipping it does not block progress.
  const mcpInstallSection = (
    <div className="space-y-3">
      <h4 className="text-sm font-medium">Optional: install the Hatchet MCP</h4>
      <CodeHighlighter
        className={codeBlockClass}
        code="hatchet mcp install"
        language="shell"
        copy
      />
    </div>
  );

  // Once the first task run completes, link out to that specific run and to
  // the worker that executed it (opened in a new tab so the flow is not lost).
  const completedRunLinkClass = cn(
    'inline-flex w-fit items-center gap-1 text-sm underline hover:text-foreground',
    focusRing,
  );
  const completedRunLinks =
    progress.completedRun && tenantId ? (
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
        <Link
          to={appRoutes.tenantRunRoute.to}
          params={{ tenant: tenantId, run: progress.completedRun.runId }}
          target="_blank"
          rel="noopener noreferrer"
          className={completedRunLinkClass}
        >
          View the task run
          <ExternalLinkIcon className="size-3" />
        </Link>
        {progress.completedRun.workerId ? (
          <Link
            to={appRoutes.tenantWorkerRoute.to}
            params={{
              tenant: tenantId,
              worker: progress.completedRun.workerId,
            }}
            target="_blank"
            rel="noopener noreferrer"
            className={completedRunLinkClass}
          >
            View the worker
            <ExternalLinkIcon className="size-3" />
          </Link>
        ) : null}
      </div>
    ) : null;

  const stepContent: Record<StepKey, ReactNode> = {
    usecase: (
      <>
        <div className="space-y-1">
          <h3 className="text-sm font-medium">What are you building?</h3>
          <p className="text-sm text-muted-foreground">
            We'll customize your prompt for getting your coding agent onboarded.
          </p>
        </div>
        <div className="space-y-2">
          <p className="text-sm font-medium">SDK</p>
          <SdkSwitcher value={sdk} onChange={onSdkChange} />
        </div>
        {path === 'manual' ? (
          <div className="space-y-2">
            <p className="text-sm font-medium">Template</p>
            <RadioGroup
              value={useCase}
              onValueChange={(value) =>
                onUseCaseChange(value as AvailableUseCaseKey)
              }
              className="grid-cols-1 gap-3 lg:grid-cols-2"
            >
              {Object.values(availableUseCases).map((option) => (
                <RadioGroupCardItem
                  key={option.value}
                  value={option.value}
                  className={brandCardClass}
                >
                  <div>
                    <span className="block text-sm font-medium">
                      {option.label}
                    </span>
                    <span className="mt-1 block text-sm text-muted-foreground">
                      {option.description}
                    </span>
                  </div>
                </RadioGroupCardItem>
              ))}
            </RadioGroup>
          </div>
        ) : (
          <>
            <div className="space-y-2">
              <div>
                <p id="onboarding-patterns" className="text-sm font-medium">
                  Patterns
                </p>
                <p className="text-sm text-muted-foreground">
                  Pick any that apply. Your agent will build with these Hatchet
                  features.
                </p>
              </div>
              {/* Toggle buttons rather than a radio group: a use case often
                  combines patterns (a cron job that kicks off a pipeline). */}
              <div
                role="group"
                aria-labelledby="onboarding-patterns"
                className="grid grid-cols-1 gap-3 lg:grid-cols-2"
              >
                {agentPatternOrder.map((key) => {
                  const selected = patterns.includes(key);
                  return (
                    <button
                      key={key}
                      type="button"
                      aria-pressed={selected}
                      onClick={() =>
                        onPatternsChange(
                          selected
                            ? patterns.filter((p) => p !== key)
                            : [...patterns, key],
                        )
                      }
                      className={cn(
                        'rounded-lg border p-4 text-left',
                        focusRing,
                        selected
                          ? 'border-brand bg-brand/10'
                          : 'border-border/50 bg-muted/20 hover:border-brand/50',
                      )}
                    >
                      <span className="block text-sm font-medium">
                        {agentPatterns[key].label}
                      </span>
                      <span className="mt-1 block text-sm text-muted-foreground">
                        {agentPatterns[key].description}
                      </span>
                    </button>
                  );
                })}
              </div>
            </div>
            <div className="space-y-2">
              <div>
                <label
                  htmlFor="onboarding-description"
                  className="text-sm font-medium"
                >
                  Describe your use case
                </label>
                <p className="text-sm text-muted-foreground">
                  Required. The more specific you are, the better the first
                  result.
                </p>
              </div>
              <Textarea
                id="onboarding-description"
                value={description}
                onChange={(e) => onDescriptionChange(e.target.value)}
                placeholder="e.g. process uploaded CSVs and email a summary when done"
                rows={3}
                className={focusRing}
              />
            </div>
          </>
        )}
      </>
    ),
    setup: (
      <>
        <div className="space-y-1">
          <h3 className="text-sm font-medium">Set up the CLI</h3>
          <p className="text-sm text-muted-foreground">
            Install the CLI and connect a profile to this tenant.
          </p>
        </div>
        <div className="space-y-3">
          <h4 className="text-sm font-medium">Install the CLI</h4>
          <Tabs
            value={installMethod}
            onValueChange={(value) => setInstallMethod(value as InstallMethod)}
            className="w-full"
          >
            <TabsList className={cn('mt-2', segmentedTabsListClass)}>
              <TabsTrigger
                value={installMethodOptions.native.value}
                className={segmentedTabsTriggerClass}
              >
                curl
              </TabsTrigger>
              <TabsTrigger
                value={installMethodOptions.homebrew.value}
                className={segmentedTabsTriggerClass}
              >
                Homebrew
              </TabsTrigger>
            </TabsList>
            <TabsContent
              value={installMethodOptions.native.value}
              className={`mt-4 space-y-3 rounded-sm ${focusRing}`}
            >
              <p className="text-sm">
                <b>MacOS, Linux, WSL</b>
              </p>
              <CodeHighlighter
                className={codeBlockClass}
                code={`curl -fsSL https://install.hatchet.run/install.sh | bash`}
                language="shell"
                copy
              />
            </TabsContent>
            <TabsContent
              value={installMethodOptions.homebrew.value}
              className={`mt-4 space-y-3 rounded-sm ${focusRing}`}
            >
              <p className="text-sm">
                <b>MacOS</b>
              </p>
              <CodeHighlighter
                className={codeBlockClass}
                code={`brew install hatchet-dev/hatchet/hatchet --cask`}
                language="shell"
                copy
              />
            </TabsContent>
          </Tabs>
          <p className="text-sm">Confirm it installed:</p>
          <CodeHighlighter
            className={codeBlockClass}
            code={`hatchet --version`}
            language="shell"
            copy
          />
        </div>
        <div className="space-y-3 border-t border-border/50 pt-4">
          <h4 className="text-sm font-medium">Set up your profile</h4>
          <p className="text-sm text-muted-foreground">
            A profile connects the CLI to this tenant.
          </p>
          {authDisabled ? (
            <>
              <p className="text-sm">
                Auth is disabled on this instance, so use the built-in token
                below.
              </p>
              <CodeHighlighter
                className={codeBlockClass}
                code={`hatchet profile add --name "${escapeForDoubleQuotes(
                  profileName,
                )}" --token "${authDisabledToken ?? '<token>'}"`}
                language="shell"
                copy
              />
            </>
          ) : (
            <>
              <div className="flex flex-wrap items-center gap-3">
                <Button
                  variant="outline"
                  size="default"
                  className={cn('w-fit gap-2', focusRing)}
                  onClick={onGenerateProfileToken}
                  disabled={isGeneratingProfileToken || !canGenerateToken}
                >
                  {isGeneratingProfileToken && <Spinner />}
                  Generate a token for this command
                </Button>
                {profileToken && (
                  <span className="text-xs text-muted-foreground">
                    This token is only shown once. Copy it now.
                  </span>
                )}
              </div>
              {profileTokenError && (
                <div className="text-sm text-red-500">{profileTokenError}</div>
              )}
              {hasApiToken && !profileToken && (
                // An existing token lets the step pass, but the API cannot
                // tell whether this machine has a CLI profile for it.
                <p className="text-xs text-muted-foreground">
                  This tenant already has an API token. If this machine does not
                  have a CLI profile for it yet, generate a token above and run
                  the command it gives you.
                </p>
              )}
              {profileToken && (
                <>
                  <p className="text-sm">Then run:</p>
                  <CodeHighlighter
                    className={codeBlockClass}
                    code={`hatchet profile add --name "${escapeForDoubleQuotes(
                      profileName,
                    )}" --token "${escapeForDoubleQuotes(profileToken)}"`}
                    language="shell"
                    copy
                  />
                </>
              )}
            </>
          )}
        </div>
      </>
    ),
    path: (
      <>
        <div className="space-y-1">
          <h3 className="text-sm font-medium">Choose your preferred setup</h3>
        </div>
        <div className="rounded-lg border border-brand/60 bg-brand/5 p-4 space-y-3">
          <div className="flex items-start gap-3">
            <Bot className="mt-0.5 size-6 shrink-0 text-brand" />
            <div className="space-y-1">
              <h4 className="text-sm font-medium">
                With your coding agent (recommended)
              </h4>
              <p className="text-sm text-muted-foreground">
                Let your coding agent scaffold, run, and debug for you.
              </p>
            </div>
          </div>
          <Button
            variant="default"
            size="sm"
            className={cn('w-fit gap-2', focusRing)}
            onClick={() => choosePath('agent')}
          >
            Set up my agent
            <ChevronRightIcon className="size-3" />
          </Button>
        </div>
        <div className="rounded-lg border border-border/50 bg-muted/20 p-4 space-y-3">
          <div className="flex items-start gap-3">
            <Terminal className="mt-0.5 size-5 shrink-0 text-muted-foreground" />
            <div className="space-y-1">
              <h4 className="text-sm font-medium">Manual setup</h4>
              <p className="text-sm text-muted-foreground">
                Scaffold a project, start a worker, and run a task yourself.
              </p>
            </div>
          </div>
          <Button
            variant="secondary"
            size="sm"
            className={cn('w-fit gap-2', focusRing)}
            onClick={() => choosePath('manual')}
          >
            Manual setup
          </Button>
        </div>
      </>
    ),
    runagent: (
      <>
        {mcpInstallSection}
        <div className="space-y-1 border-t border-border/50 pt-4">
          <h3 className="text-sm font-medium">Run your agent</h3>
          <p className="text-sm text-muted-foreground">
            Here's a sample prompt for your coding agent. You can customize it
            to fit your needs.
          </p>
        </div>
        <CodeHighlighter
          className={`${codeBlockClass} whitespace-pre-wrap`}
          code={generatedPrompt}
          language="text"
          maxHeight="220px"
          copy
        />
        <p className="text-xs text-muted-foreground">
          No need to refresh, we're watching for your worker and run.
        </p>
        <StatusRow
          ready={progress.workerConnected}
          done="Worker connected"
          waiting="Waiting for a worker to connect..."
        />
        <StatusRow
          ready={progress.runCompleted}
          done="Task run completed"
          waiting="Waiting for a task to run..."
        />
        {completedRunLinks}
      </>
    ),
    runtask: (
      <>
        <div className="space-y-1">
          <h3 className="text-sm font-medium">Run a task</h3>
          <p className="text-sm text-muted-foreground">
            Scaffold a project, start a worker, then trigger a run and watch it
            complete.
          </p>
        </div>
        <p className="text-sm">Scaffold the project:</p>
        <CodeHighlighter
          className={codeBlockClass}
          code={scaffoldCommand({ useCase, language: sdk })}
          language="shell"
          copy
        />
        <p className="text-sm">Start a dev worker:</p>
        <CodeHighlighter
          className={codeBlockClass}
          code={workerDevCommand(profileName)}
          language="shell"
          copy
        />
        <StatusRow
          ready={progress.workerConnected}
          done="Worker connected"
          waiting="Waiting for the worker to connect..."
        />
        <p className="text-sm">Trigger a run:</p>
        <CodeHighlighter
          className={codeBlockClass}
          code={triggerCommand(useCase, profileName)}
          language="shell"
          copy
        />
        <StatusRow
          ready={progress.runCompleted}
          done="Run completed"
          waiting="Waiting for the run to complete..."
        />
        {completedRunLinks}
      </>
    ),
    finish: (
      <>
        <div className="space-y-1">
          <h3 className="text-sm font-medium">You're set up</h3>
          <p className="text-sm text-muted-foreground">
            Your worker is connected and running tasks against this tenant.
          </p>
        </div>
        <div className="space-y-3">
          <h4 className="text-sm font-medium">Learn more</h4>
          <ul className="space-y-2">
            {relevantDocs({ sdk, useCase, patterns, path }).map((doc) => (
              <li key={doc.href}>
                <a
                  href={doc.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex w-fit items-center gap-1 text-sm underline hover:text-foreground"
                >
                  {doc.label}
                  <ExternalLinkIcon className="size-3" />
                </a>
              </li>
            ))}
          </ul>
        </div>
      </>
    ),
  };

  const currentIndex = sequence.indexOf(currentStep);
  const nextStep = sequence[currentIndex + 1];

  const goBack = () => {
    const prev = sequence[currentIndex - 1];
    if (prev) {
      onNavigate({ path, step: prev });
    }
  };

  const goNext = () => {
    if (!nextStep) {
      return;
    }
    if (nextStep === 'runagent') {
      onPromptGenerated(patterns, sdk);
    }
    goTo(nextStep);
  };

  // Each step's completion gate. Token and run gates are checked against the
  // API so they survive a refresh.
  const gateMet = (target: StepKey) => {
    switch (target) {
      case 'usecase':
        // The agent prompt is built around the developer's own description, so
        // it is required; the manual path only needs its (defaulted) template.
        return path === 'manual' || description.trim().length > 0;
      case 'setup':
        return hasApiToken;
      case 'runagent':
      case 'runtask':
        return progress.runCompleted;
      default:
        return true;
    }
  };
  const stepGateMet = gateMet(currentStep);
  // The step rail can jump backwards freely, but not past the first step
  // whose gate is unmet (otherwise "Run agent" is reachable with no
  // description and a prompt that claims setup is done).
  const firstUnmet = sequence.findIndex((candidate) => !gateMet(candidate));
  const furthestIndex = firstUnmet === -1 ? sequence.length - 1 : firstUnmet;

  // The path selector advances by picking a card; finish keeps its own gated
  // CTA. Every other step advances via Next once its gate is met.
  const showNext =
    currentStep !== 'path' &&
    currentStep !== 'finish' &&
    Boolean(nextStep) &&
    stepGateMet;

  return (
    <div className="space-y-4">
      <ol className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {sequence.map((railStep, index) => (
          <li key={railStep}>
            <button
              type="button"
              onClick={() => goTo(railStep)}
              disabled={index > furthestIndex}
              aria-current={railStep === currentStep ? 'step' : undefined}
              className={cn(
                'font-medium transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:text-muted-foreground',
                focusRing,
                railStep === currentStep && 'text-foreground font-semibold',
                railStep !== currentStep &&
                  index < currentIndex &&
                  'text-foreground',
              )}
            >
              {index + 1}. {stepRailLabels[railStep]}
            </button>
          </li>
        ))}
      </ol>
      <div className="rounded-md p-4 bg-muted/20 ring-1 ring-border/50 ring-inset space-y-4">
        {stepContent[currentStep]}
        <div className="flex items-center justify-between border-t border-border/50 pt-4">
          <div className="flex items-center gap-2">
            <HelpDropdown
              icon={<LifeBuoy className="size-5 text-foreground" />}
              align="start"
              side="top"
              className="text-muted-foreground"
            />
            {currentIndex > 0 && (
              <Button
                variant="ghost"
                size="sm"
                className={`gap-1 text-muted-foreground ${focusRing}`}
                onClick={goBack}
              >
                <ChevronLeftIcon className="size-3" />
                Back
              </Button>
            )}
            {path === 'manual' && currentStep !== 'finish' && (
              <Button
                variant="ghost"
                size="sm"
                className={`text-muted-foreground underline ${focusRing}`}
                onClick={() => choosePath('agent')}
              >
                Switch to the agent setup
              </Button>
            )}
          </div>
          {showNext && (
            <Button
              variant="default"
              size="sm"
              className={cn('gap-1', focusRing)}
              onClick={goNext}
            >
              Next
              <ChevronRightIcon className="size-3" />
            </Button>
          )}
          {currentStep === 'finish' && (
            <Button
              variant="default"
              size="sm"
              className={cn('gap-1', focusRing)}
              disabled={!progress.runCompleted}
              hoverText={
                progress.runCompleted
                  ? undefined
                  : 'Waiting for a worker to connect and execute a task.'
              }
              onClick={onFinish}
            >
              Start exploring
              <ChevronRightIcon className="size-3" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
