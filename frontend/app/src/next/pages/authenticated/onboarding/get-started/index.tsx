import { Loading } from '@/components/ui/loading';
import { Button } from '@/components/ui/button';
import { Step, Steps } from '@/components/v1/ui/steps';
import { DefaultOnboardingAuth } from './platforms/defaults/default-onboarding-auth';
import { DefaultOnboardingWorkflow } from './platforms/defaults/default-onboarding-workflow';
import { useState } from 'react';
import { WorkerListener } from './platforms/defaults/default-worker-listener';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { CodeHighlighter } from '@/components/ui/code-highlighter';
import { useTenantDetails } from '@/next/hooks/use-tenant';
import useUser from '@/next/hooks/use-user';
import { Link } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';

export default function GetStarted() {
  const { data: user, memberships } = useUser();
  const { tenant: currTenant } = useTenantDetails();

  const [quickstartClonedOpen, setQuickstartClonedOpen] = useState(true);
  const [quickstartCloned, setQuickstartCloned] = useState(false);
  const [tokenGeneratedOpen, setTokenGeneratedOpen] = useState(false);
  const [tokenGenerated, setTokenGenerated] = useState(false);
  const [workerStartedOpen, setWorkerStartedOpen] = useState(false);
  const [workerStarted, setWorkerStarted] = useState(false);
  const [workflowTriggeredOpen, setWorkflowTriggeredOpen] = useState(false);

  const [language, setLanguage] = useState('python');
  const [packageManager, setPackageManager] = useState('npm');

  if (!user || !memberships || !currTenant) {
    return <Loading />;
  }

  return (
    <div className="flex flex-col items-center w-full h-full overflow-auto">
      <div className="container mx-auto px-4 py-8 lg:px-8 lg:py-12 max-w-4xl">
        <div className="flex flex-col justify-center space-y-4">
          <div className="flex flex-row justify-between mt-10">
            <h1 className="text-3xl font-bold">Quickstart</h1>
            <Link to={{ pathname: ROUTES.runs.list(currTenant.metadata.id) }}>
              <Button variant="outline">Skip Quickstart</Button>
            </Link>
          </div>

          <p className="text-gray-600 dark:text-gray-300 text-sm">
            Get started by deploying your first worker.
          </p>

          <Steps className="mt-6">
            <Step
              title="Clone our quickstart repository"
              open={quickstartClonedOpen}
              setOpen={setQuickstartClonedOpen}
            >
              <div className="grid gap-4">
                <div className="text-sm text-muted-foreground">
                  Want to import Hatchet into an existing project? Check out our
                  <a
                    href="https://docs.hatchet.run/home/setup"
                    target="_blank"
                    rel="noreferrer"
                    className="text-blue-500"
                  >
                    {' '}
                    documentation
                  </a>
                  .
                </div>
                <Tabs
                  value={language}
                  onValueChange={setLanguage}
                  className="w-full"
                >
                  <TabsList className="mt-2">
                    <TabsTrigger value="python">Python</TabsTrigger>
                    <TabsTrigger value="typescript">Typescript</TabsTrigger>
                    <TabsTrigger value="go">Go</TabsTrigger>
                  </TabsList>
                  <TabsContent value="python" className="mt-4 space-y-3">
                    <div className="text-sm text-muted-foreground mb-4">
                      Clone the repository and install dependencies:
                    </div>
                    <CodeHighlighter
                      code={`git clone https://github.com/hatchet-dev/hatchet-python-quickstart.git\ncd hatchet-python-quickstart\npoetry install`}
                      copyCode={`git clone https://github.com/hatchet-dev/hatchet-python-quickstart.git\ncd hatchet-python-quickstart\npoetry install`}
                      language="shell"
                      copy
                    />
                  </TabsContent>
                  <TabsContent value="typescript" className="mt-4 space-y-3">
                    <div className="text-sm text-muted-foreground mb-4">
                      Clone the repository and install dependencies:
                    </div>
                    <CodeHighlighter
                      code={`git clone https://github.com/hatchet-dev/hatchet-typescript-quickstart.git\ncd hatchet-typescript-quickstart`}
                      copyCode={`git clone https://github.com/hatchet-dev/hatchet-typescript-quickstart.git\ncd hatchet-typescript-quickstart`}
                      language="shell"
                      copy
                    />
                    <Tabs
                      value={packageManager}
                      onValueChange={setPackageManager}
                    >
                      <TabsList className="mt-2">
                        <TabsTrigger value="npm">npm</TabsTrigger>
                        <TabsTrigger value="pnpm">pnpm</TabsTrigger>
                        <TabsTrigger value="yarn">yarn</TabsTrigger>
                      </TabsList>
                      <TabsContent value="npm" className="mt-3">
                        <CodeHighlighter
                          code="npm install"
                          copyCode="npm install"
                          language="shell"
                          copy
                        />
                      </TabsContent>
                      <TabsContent value="pnpm" className="mt-3">
                        <CodeHighlighter
                          code="pnpm install"
                          copyCode="pnpm install"
                          language="shell"
                          copy
                        />
                      </TabsContent>
                      <TabsContent value="yarn" className="mt-3">
                        <CodeHighlighter
                          code="yarn install"
                          copyCode="yarn install"
                          language="shell"
                          copy
                        />
                      </TabsContent>
                    </Tabs>
                  </TabsContent>
                  <TabsContent value="go" className="mt-4 space-y-3">
                    <div className="text-sm text-muted-foreground mb-4">
                      Clone the repository and install dependencies:
                    </div>
                    <CodeHighlighter
                      code={`git clone https://github.com/hatchet-dev/hatchet-go-quickstart.git\ncd hatchet-go-quickstart\ngo mod tidy`}
                      copyCode={`git clone https://github.com/hatchet-dev/hatchet-go-quickstart.git\ncd hatchet-go-quickstart\ngo mod tidy`}
                      language="shell"
                      copy
                    />
                  </TabsContent>
                </Tabs>
                <Button
                  onClick={() => {
                    setQuickstartCloned(true);
                    setQuickstartClonedOpen(false);
                    setTokenGeneratedOpen(true);
                  }}
                  className="w-fit"
                  variant="outline"
                >
                  Continue
                </Button>
              </div>
            </Step>
            <Step
              title="Generate an API token"
              open={tokenGeneratedOpen}
              setOpen={setTokenGeneratedOpen}
              disabled={!quickstartCloned}
            >
              <div className="grid gap-4">
                <DefaultOnboardingAuth
                  tenantId={currTenant.metadata.id}
                  tokenGenerated={() => {
                    setTokenGenerated(true);
                    setTokenGeneratedOpen(false);
                    setWorkerStartedOpen(true);
                  }}
                />
              </div>
            </Step>
            <Step
              title="Start your worker"
              open={workerStartedOpen}
              setOpen={setWorkerStartedOpen}
              disabled={!tokenGenerated}
            >
              <div className="grid gap-4">
                <Tabs value={language} onValueChange={setLanguage}>
                  <TabsList className="mt-2">
                    <TabsTrigger value="python">Python</TabsTrigger>
                    <TabsTrigger value="typescript">Typescript</TabsTrigger>
                    <TabsTrigger value="go">Go</TabsTrigger>
                  </TabsList>
                  <TabsContent value="python" className="mt-4">
                    <CodeHighlighter
                      code="poetry run python src/worker.py"
                      copyCode="poetry run python src/worker.py"
                      language="shell"
                      copy
                    />
                  </TabsContent>
                  <TabsContent value="typescript" className="mt-4">
                    <Tabs
                      value={packageManager}
                      onValueChange={setPackageManager}
                    >
                      <TabsList className="mt-2">
                        <TabsTrigger value="npm">npm</TabsTrigger>
                        <TabsTrigger value="pnpm">pnpm</TabsTrigger>
                        <TabsTrigger value="yarn">yarn</TabsTrigger>
                      </TabsList>
                      <TabsContent value="npm" className="mt-3">
                        <CodeHighlighter
                          code="npm run start"
                          copyCode="npm run start"
                          language="shell"
                          copy
                        />
                      </TabsContent>
                      <TabsContent value="pnpm" className="mt-3">
                        <CodeHighlighter
                          code="pnpm run start"
                          copyCode="pnpm run start"
                          language="shell"
                          copy
                        />
                      </TabsContent>
                      <TabsContent value="yarn" className="mt-3">
                        <CodeHighlighter
                          code="yarn run start"
                          copyCode="yarn run start"
                          language="shell"
                          copy
                        />
                      </TabsContent>
                    </Tabs>
                  </TabsContent>
                  <TabsContent value="go" className="mt-4">
                    <CodeHighlighter
                      code="go run cmd/worker/main.go"
                      copyCode="go run cmd/worker/main.go"
                      language="shell"
                      copy
                    />
                  </TabsContent>
                </Tabs>
                <WorkerListener
                  tenantId={currTenant.metadata.id}
                  setWorkerConnected={() => {
                    setWorkerStarted(true);
                  }}
                />
                Waiting for worker to connect...
                <Button
                  onClick={() => {
                    setWorkerStartedOpen(false);
                    setWorkflowTriggeredOpen(true);
                  }}
                  className="w-fit"
                  variant="outline"
                  disabled={!workerStarted}
                >
                  Continue
                </Button>
              </div>
            </Step>
            <Step
              title="Trigger a workflow"
              open={workflowTriggeredOpen}
              setOpen={setWorkflowTriggeredOpen}
              disabled={!workerStarted}
            >
              <div className="grid gap-4">
                <DefaultOnboardingWorkflow tenantId={currTenant.metadata.id} />
              </div>
            </Step>
          </Steps>
        </div>
      </div>
    </div>
  );
}
