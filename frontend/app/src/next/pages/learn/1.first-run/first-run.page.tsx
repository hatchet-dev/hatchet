import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/next/components/ui/card';
import { MembersProvider } from '@/next/hooks/use-members';
import { TwoColumnLayout } from '@/next/components/layouts/two-column.layout';
import { GithubCode } from '@/next/components/ui/code/github-code';

export default function OnboardingFirstRunPage() {
  return (
    <MembersProvider>
      <OnboardingFirstRunContent />
    </MembersProvider>
  );
}

function OnboardingFirstRunContent() {
  return (
    <TwoColumnLayout
      left={
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>1. Setup Your Environment</CardTitle>
              <CardDescription>
                Choose your technology stack and install the necessary
                dependencies.
              </CardDescription>
            </CardHeader>
            <CardContent>{/* Technology selection will go here */}</CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>2. Writing Your First Task</CardTitle>
              <CardDescription>
                Define a task that performs a specific action in your workflow.
              </CardDescription>
            </CardHeader>
            <CardContent>{/* Task writing content will go here */}</CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>3. Registering Your First Worker</CardTitle>
              <CardDescription>
                Create a worker to execute your tasks.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {/* Worker registration content will go here */}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>4. Running Your Task</CardTitle>
              <CardDescription>
                Trigger your task and see it in action.
              </CardDescription>
            </CardHeader>
            <CardContent>{/* Task running content will go here */}</CardContent>
          </Card>
        </div>
      }
      right={
        <>
          <div className="space-y-2">
            <h3 className="text-sm font-medium">Client</h3>
            <GithubCode
              repo="hatchet-dev/hatchet-typescript-quickstart"
              path="src/hatchet-client.ts"
              language="typescript"
              showLineNumbers={true}
            />
            <GithubCode
              repo="hatchet-dev/hatchet-typescript-quickstart"
              path="src/workflows/first-workflow.ts"
              language="typescript"
              showLineNumbers={true}
            />
            <GithubCode
              repo="hatchet-dev/hatchet-typescript-quickstart"
              path="src/worker.ts"
              language="typescript"
              showLineNumbers={true}
            />
            <GithubCode
              repo="hatchet-dev/hatchet-typescript-quickstart"
              path="src/run.ts"
              language="typescript"
              showLineNumbers={true}
            />
          </div>
        </>
      }
    />
  );
}
