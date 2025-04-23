import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/next/components/ui/card';
import { MembersProvider } from '@/next/hooks/use-members';
import { Code } from '@/next/components/ui/code/code';
import { TwoColumnLayout } from '@/next/components/layouts/two-column.layout';

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
            <Code
              title="/client.ts"
              language="typescript"
              value={`// Initialize the Hatchet client
import { Hatchet } from '@hatchet-dev/typescript-sdk';

const hatchet = Hatchet.init();`}
              showLineNumbers
            />
          </div>

          <div className="space-y-2">
            <h3 className="text-sm font-medium">Task</h3>
            <Code
              title="/task.ts"
              language="typescript"
              value={`// Define your task
const task = hatchet.task({
  name: 'hello-world',
  fn: (input: { name: string }) => {
    return {
      message: \`Hello, \${input.name}!\`
    };
  },
});`}
              showLineNumbers
            />
          </div>

          <div className="space-y-2">
            <h3 className="text-sm font-medium">Worker</h3>
            <Code
              title="/worker.ts"
              language="typescript"
              value={`// Create a worker to run your task
const worker = hatchet.worker({
  name: 'hello-world-worker',
  tasks: [task],
});`}
              showLineNumbers
            />
          </div>

          <div className="space-y-2">
            <h3 className="text-sm font-medium">Trigger</h3>
            <Code
              title="/trigger.ts"
              language="typescript"
              value={`// Trigger your task
await task.run({
  name: 'World'
});`}
              showLineNumbers
            />
          </div>
        </>
      }
    />
  );
}
