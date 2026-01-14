import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardTitle,
  CardHeader,
  CardContent,
} from '@/components/v1/ui/card';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/v1/ui/dropdown-menu';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { SecretCopier } from '@/components/v1/ui/secret-copier';
import { Separator } from '@/components/v1/ui/separator';
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from '@/components/v1/ui/tabs';
import { useAnalytics } from '@/hooks/use-analytics';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, { CreateAPITokenRequest, queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import {
  ChevronRightIcon,
  ChevronDownIcon,
  CheckIcon,
} from '@radix-ui/react-icons';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import { useState, useEffect, useRef } from 'react';
import { RiDiscordFill, RiGithubFill, RiLink } from 'react-icons/ri';

const EXPIRES_IN_OPTIONS = {
  '3 months': `${3 * 30 * 24 * 60 * 60}s`,
  '1 year': `${365 * 24 * 60 * 60}s`,
  '100 years': `${100 * 365 * 24 * 60 * 60}s`,
};

const onboardingSVG = (
  <svg
    width="12"
    height="12"
    viewBox="0 0 12 12"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    className="bottom-[1px] relative"
  >
    <path
      d="M10.499 1.5V2.5C10.499 7.3137 7.81275 9.5 4.49902 9.5L3.5482 9.49995C3.65395 7.9938 4.1227 7.0824 5.34695 5.99945C5.94895 5.4669 5.89825 5.15945 5.60145 5.33605C3.55965 6.5508 2.54557 8.1931 2.50059 10.8151L2.49902 11H1.49902C1.49902 10.3187 1.55688 9.69985 1.6719 9.1341C1.55665 8.48705 1.49902 7.6088 1.49902 6.5C1.49902 3.73857 3.7376 1.5 6.499 1.5C7.499 1.5 8.499 2 10.499 1.5Z"
      fill="hsl(var(--brand))"
    />
  </svg>
);

const workflowSteps = {
  settingUp: 'Setting Up',
  chooseToken: 'Set env vars',
  runWorker: 'Run worker',
  viewTask: 'Launch task',
};

const workflowLanguages = {
  python: 'Python',
  typescript: 'TypeScript',
  go: 'Go',
};

export default function Overview() {
  const { tenantId } = useCurrentTenantId();
  const { currentUser } = useCurrentUser();
  const navigate = useNavigate();
  const { capture } = useAnalytics();
  const [tokenName, setTokenName] = useState('');
  const [expiresIn, setExpiresIn] = useState(EXPIRES_IN_OPTIONS['100 years']);
  const [generatedToken, setGeneratedToken] = useState<string | undefined>();
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [selectedTab, setSelectedTab] = useState<
    'settingUp' | 'chooseToken' | 'runWorker' | 'viewTask'
  >('settingUp');
  const [language, setLanguage] = useState<string>('python');
  const hasTrackedWorkerConnection = useRef(false);

  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  // Track page view on mount
  useEffect(() => {
    capture('onboarding_overview_viewed', {
      tenant_id: tenantId,
      user_email: currentUser?.email,
    });
  }, [capture, tenantId, currentUser?.email]);

  const createTokenMutation = useMutation({
    mutationKey: ['api-token:create', tenantId],
    mutationFn: async (data: CreateAPITokenRequest) => {
      const res = await api.apiTokenCreate(tenantId, data);
      return res.data;
    },
    onSuccess: (data) => {
      setGeneratedToken(data.token);
      setShowTokenDialog(true);
      // Track token generation
      capture('onboarding_token_generated', {
        tenant_id: tenantId,
        user_email: currentUser?.email,
        token_name: tokenName,
        expires_in: expiresIn,
      });
      // Reset form
      setTokenName('');
    },
    onError: handleApiError,
  });

  const handleGenerateToken = () => {
    if (!tokenName.trim()) {
      setFieldErrors({ name: 'Name is required' });
      return;
    }
    createTokenMutation.mutate({
      name: tokenName,
      expiresIn: expiresIn,
    });
  };

  // Poll for workers when on the "Run worker" tab
  const workersQuery = useQuery({
    ...queries.workers.list(tenantId),
    enabled: selectedTab === 'runWorker',
    refetchInterval: selectedTab === 'runWorker' ? 2000 : false, // Poll every 2 seconds
  });

  const hasActiveWorker =
    workersQuery.data?.rows && workersQuery.data.rows.length > 0;

  // Track worker connection (only once)
  useEffect(() => {
    if (hasActiveWorker && !hasTrackedWorkerConnection.current) {
      capture('onboarding_worker_connected', {
        tenant_id: tenantId,
        user_email: currentUser?.email,
      });
      hasTrackedWorkerConnection.current = true;
    }
  }, [hasActiveWorker, capture, tenantId, currentUser?.email]);

  return (
    <div className="flex h-full w-full flex-col space-y-24 lg:p-6">
      {/* Header section */}
      <div className="grid gap-x-2 gap-y-6 grid-cols-1 items-start lg:grid-cols-[1fr_auto]">
        <div className="flex items-center gap-6 flex-wrap">
          <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
          {/* <OnboardingWidget steps={4} currentStep={1} label="Steps completed" /> */}
        </div>
        <p className="text-muted-foreground text-balance">
          Complete your onboarding on this page
        </p>
      </div>
      {/* Create API token section */}
      <div>
        <span className="inline-flex items-baseline gap-5">
          <h2 className="text-md">Create API token</h2>
          <span className="text-[10px] font-mono tracking-widest uppercase inline-flex items-center gap-1.5 text-brand">
            {onboardingSVG}
            Onboarding step
          </span>
        </span>
        <Separator className="my-4 bg-border/50" flush />
        <div className="grid gap-4 items-end grid-flow-row lg:[grid-template-columns:1fr_1fr_auto_1fr] lg:grid-flow-col">
          <div className="grid gap-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              type="text"
              required={true}
              autoCapitalize="none"
              autoCorrect="off"
              placeholder="My Token"
              value={tokenName}
              onChange={(e) => {
                setTokenName(e.target.value);
                setFieldErrors({});
              }}
              disabled={createTokenMutation.isPending}
            />
            {fieldErrors.name && (
              <div className="text-sm text-red-500">{fieldErrors.name}</div>
            )}
          </div>
          <div className="grid gap-2">
            <Label htmlFor="expiresIn">Expires In</Label>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="outline"
                  size="default"
                  className="flex justify-between data-[state=open]:bg-muted"
                  disabled={createTokenMutation.isPending}
                >
                  {Object.entries(EXPIRES_IN_OPTIONS).find(
                    ([, value]) => value === expiresIn,
                  )?.[0] || 'Select an option'}
                  <ChevronDownIcon className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-[160px]">
                {Object.entries(EXPIRES_IN_OPTIONS).map(([label, value]) => (
                  <DropdownMenuItem
                    key={value}
                    onClick={() => {
                      setExpiresIn(value);
                      capture('onboarding_token_expiration_selected', {
                        tenant_id: tenantId,
                        user_email: currentUser?.email,
                        expiration: label,
                      });
                    }}
                  >
                    {label}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <Separator orientation="vertical" className="hidden lg:block" />
          <div className="grid gap-2 justify-self-start">
            <Button
              variant="default"
              size="default"
              onClick={handleGenerateToken}
              disabled={createTokenMutation.isPending}
            >
              {createTokenMutation.isPending && <Spinner />}
              Generate Token
            </Button>
          </div>
        </div>
      </div>
      {/* Learn the workflow section */}
      <div>
        <span className="inline-flex items-baseline gap-5">
          <h2 className="text-md">Learn the workflow</h2>
          <span className="text-[10px] font-mono tracking-widest uppercase inline-flex items-center gap-1.5 text-brand">
            {onboardingSVG}
            Onboarding step
          </span>
        </span>
        <Separator className="my-4 bg-border/50" flush />
        <Tabs
          value={selectedTab}
          onValueChange={(value) => {
            const newTab = value as
              | 'settingUp'
              | 'chooseToken'
              | 'runWorker'
              | 'viewTask';
            setSelectedTab(newTab);
            capture('onboarding_tab_changed', {
              tenant_id: tenantId,
              user_email: currentUser?.email,
              tab: workflowSteps[newTab],
            });
          }}
          className="w-full rounded-md px-6 pb-6 bg-muted/20 ring-1 ring-border/50 ring-inset"
        >
          <TabsList className="grid w-full grid-flow-col rounded-none bg-transparent p-0 pb-6 justify-start gap-6 h-auto ">
            {Object.entries(workflowSteps).map(([key, value], index) => (
              <TabsTrigger
                key={key}
                value={key}
                className={
                  'text-xs text-muted-foreground rounded-none pt-2.5 px-1 font-medium border-t border-transparent hover:border-border bg-transparent data-[state=active]:border-primary/50 data-[state=active]:bg-transparent data-[state=active]:shadow-none'
                }
              >
                <div className="flex items-center gap-2">
                  {index + 1} {value}
                </div>
              </TabsTrigger>
            ))}
          </TabsList>

          <TabsContent value="settingUp" className="mt-0 space-y-5">
            <p> Clone the repository and install dependencies. </p>
            <Tabs
              value={language}
              onValueChange={(value) => {
                setLanguage(value);
                capture('onboarding_language_selected', {
                  tenant_id: tenantId,
                  user_email: currentUser?.email,
                  language:
                    workflowLanguages[value as keyof typeof workflowLanguages],
                });
              }}
              className="w-full"
            >
              <TabsList className="mt-2 bg-muted ring-1 ring-border/50 rounded-lg p-0 gap-0.5 dark:bg-muted/20 dark:ring-inset">
                {Object.entries(workflowLanguages).map(([key, value]) => (
                  <TabsTrigger
                    key={key}
                    value={key}
                    className="rounded-lg h-full text-muted-foreground data-[state=active]:ring-1 data-[state=active]:ring-border data-[state=active]:bg-background dark:data-[state=active]:bg-muted/70 dark:data-[state=active]:shadow-lg dark:ring-inset"
                  >
                    {value}
                  </TabsTrigger>
                ))}
              </TabsList>
              <TabsContent value="python" className="mt-4 space-y-3">
                <CodeHighlighter
                  className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
                  code={`git clone https://github.com/hatchet-dev/hatchet-python-quickstart.git\ncd hatchet-python-quickstart\npoetry install`}
                  copyCode={`git clone https://github.com/hatchet-dev/hatchet-python-quickstart.git\ncd hatchet-python-quickstart\npoetry install`}
                  language="shell"
                  copy
                />
              </TabsContent>
              <TabsContent value="typescript" className="mt-4 space-y-3">
                <CodeHighlighter
                  className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
                  code={`git clone https://github.com/hatchet-dev/hatchet-typescript-quickstart.git\ncd hatchet-typescript-quickstart`}
                  copyCode={`git clone https://github.com/hatchet-dev/hatchet-typescript-quickstart.git\ncd hatchet-typescript-quickstart`}
                  language="shell"
                  copy
                />
              </TabsContent>
              <TabsContent value="go" className="mt-4 space-y-3">
                <CodeHighlighter
                  className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
                  code={`git clone https://github.com/hatchet-dev/hatchet-go-quickstart.git\ncd hatchet-go-quickstart\ngo mod tidy`}
                  copyCode={`git clone https://github.com/hatchet-dev/hatchet-go-quickstart.git\ncd hatchet-go-quickstart\ngo mod tidy`}
                  language="shell"
                  copy
                />
              </TabsContent>
            </Tabs>
            <Button
              variant="outline"
              size="default"
              className="w-fit gap-2 bg-muted/70"
              onClick={() => setSelectedTab('chooseToken')}
            >
              Continue
              <ChevronRightIcon className="size-3 text-foreground/50" />
            </Button>
          </TabsContent>

          <TabsContent value="chooseToken" className="mt-0 space-y-5">
            <p>
              Set your API token as an environment variable. Copy the token from
              the "Create API token" section above.
            </p>
            <CodeHighlighter
              className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
              code={`export HATCHET_CLIENT_TOKEN="your-api-token"`}
              copyCode={`export HATCHET_CLIENT_TOKEN="your-api-token"`}
              language="shell"
              copy
            />
            <p>If running locally, also set:</p>
            <CodeHighlighter
              className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
              code={`export HATCHET_CLIENT_TLS_STRATEGY=none`}
              copyCode={`export HATCHET_CLIENT_TLS_STRATEGY=none`}
              language="shell"
              copy
            />
            <Button
              variant="outline"
              size="default"
              className="w-fit gap-2 bg-muted/70"
              onClick={() => setSelectedTab('runWorker')}
            >
              Continue
              <ChevronRightIcon className="size-3 text-foreground/50" />
            </Button>
          </TabsContent>
          <TabsContent value="runWorker" className="mt-0 space-y-5">
            <div className="flex items-center gap-3 rounded-lg border border-border/50 bg-muted/20 p-4">
              {hasActiveWorker ? (
                <>
                  <CheckIcon className="size-5 text-green-500" />
                  <span className="text-sm font-medium">
                    Worker is connected
                  </span>
                </>
              ) : (
                <>
                  <Spinner className="size-5" />
                  <span className="text-sm text-muted-foreground">
                    Waiting for worker...
                  </span>
                </>
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              Start your worker by running the quickstart code in your terminal.
              Once connected, you'll see a confirmation above.
            </p>
            <Button
              variant="outline"
              size="default"
              className="w-fit gap-2 bg-muted/70"
              onClick={() => setSelectedTab('viewTask')}
            >
              Continue
              <ChevronRightIcon className="size-3 text-foreground/50" />
            </Button>
          </TabsContent>
          <TabsContent value="viewTask" className="mt-0 space-y-5">
            <p className="text-sm">
              Your worker is connected and ready to process tasks. Here's what
              you can do next:
            </p>
            <div className="space-y-3">
              <Link
                to="/tenants/$tenant/runs"
                params={{ tenant: tenantId }}
                className="flex items-center gap-2 text-sm text-primary hover:underline"
              >
                <ChevronRightIcon className="size-4" />
                Trigger a task
              </Link>
              <Link
                to="/tenants/$tenant/workflows"
                params={{ tenant: tenantId }}
                className="flex items-center gap-2 text-sm text-primary hover:underline"
              >
                <ChevronRightIcon className="size-4" />
                View list of workflows
              </Link>
            </div>
            <Button
              variant="outline"
              size="default"
              className="w-fit gap-2 bg-muted/70"
              onClick={() => {
                capture('onboarding_completed', {
                  tenant_id: tenantId,
                  user_email: currentUser?.email,
                });
                navigate({
                  to: '/tenants/$tenant/runs',
                  params: { tenant: tenantId },
                });
              }}
            >
              Finish
              <CheckIcon className="size-3 text-brand" />
            </Button>
          </TabsContent>
        </Tabs>
      </div>
      {/* Support section */}
      <div className="pb-6">
        <h2 className="text-md">Support</h2>
        <Separator className="my-4 bg-border/50" flush />
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <Card
            variant="light"
            className="bg-transparent ring-1 ring-border/50 border-none"
          >
            <CardHeader className="p-4 border-b border-border/50 ">
              <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground whitespace-nowrap">
                Community
              </CardTitle>
            </CardHeader>
            <CardContent className="p-4">
              <ul className="space-y-2">
                <li>
                  <a
                    href="https://discord.gg/ZMeUafwH89"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1 text-sm text-primary/70 hover:underline hover:text-primary"
                  >
                    <RiDiscordFill className="mr-2" /> Join Discord
                  </a>
                </li>
                <li>
                  <a
                    href="https://github.com/hatchet-dev/hatchet/discussions"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1 text-sm text-primary/70 hover:underline hover:text-primary"
                  >
                    <RiGithubFill className="mr-2" /> Github Discussions
                  </a>
                </li>
              </ul>
            </CardContent>
          </Card>
          <Card
            variant="light"
            className="bg-transparent ring-1 ring-border/50 border-none"
          >
            <CardHeader className="p-4 border-b border-border/50 ">
              <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground whitespace-nowrap">
                Links
              </CardTitle>
            </CardHeader>
            <CardContent className="p-4">
              <ul className="space-y-2">
                <li>
                  <a
                    href="https://discord.gg/ZMeUafwH89"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1 text-sm text-primary/70 hover:underline hover:text-primary"
                  >
                    <RiLink className="mr-2" />
                    Documentation
                  </a>
                </li>
                <li>
                  <a
                    href="mailto:support@hatchet.run"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1 text-sm text-primary/70 hover:underline hover:text-primary"
                  >
                    <RiLink className="mr-2" />
                    Request Slack Channel
                  </a>
                </li>
                <li>
                  <a
                    href="mailto:support@hatchet.run"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1 text-sm text-primary/70 hover:underline hover:text-primary"
                  >
                    <RiLink className="mr-2" />
                    Email Support
                  </a>
                </li>
              </ul>
            </CardContent>
          </Card>
          <Card
            variant="light"
            className="bg-transparent ring-1 ring-border/50 border-none"
          >
            <CardHeader className="p-4 border-b border-border/50 ">
              <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground whitespace-nowrap">
                Office Hours
              </CardTitle>
            </CardHeader>
            <CardContent className="p-4 space-y-2">
              <span className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground">
                GMT-5 Eastern Standard Time
              </span>
              <p className="text-muted-foreground whitespace-nowrap">
                Weekdays <span className="text-primary">09:00 - 17:00</span>
              </p>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Token Success Modal */}
      {showTokenDialog && generatedToken && (
        <Dialog open={showTokenDialog} onOpenChange={setShowTokenDialog}>
          <DialogContent className="w-fit max-w-[700px]">
            <DialogHeader>
              <DialogTitle>Keep it secret, keep it safe</DialogTitle>
            </DialogHeader>
            <p className="text-sm">
              This is the only time we will show you this token. Make sure to
              copy it somewhere safe.
            </p>
            <SecretCopier
              secrets={{ HATCHET_CLIENT_TOKEN: generatedToken }}
              className="text-sm"
              maxWidth={'calc(700px - 4rem)'}
              copy
            />
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}
