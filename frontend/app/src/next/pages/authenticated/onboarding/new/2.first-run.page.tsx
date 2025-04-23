import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/next/components/ui/card';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { MembersProvider } from '@/next/hooks/use-members';
import { Code } from '@/next/components/ui/code/code';

export default function OnboardingFirstRunPage() {
  return (
    <MembersProvider>
      <OnboardingFirstRunContent />
    </MembersProvider>
  );
}

function OnboardingFirstRunContent() {
  return (
    <BasicLayout>
      <div className="flex h-[calc(100vh-4rem)] gap-4 p-4">
        {/* Left panel - Tutorial content */}
        <div className="flex-1 overflow-y-auto pr-4">
          <Card>
            <CardHeader>
              <CardTitle>Getting Started with Hatchet</CardTitle>
              <CardDescription>
                Let's create your first task and get it running in minutes.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Tutorial content will go here */}
              <div className="space-y-4">
                <h3 className="text-lg font-semibold">
                  Step 1: Choose your technology
                </h3>
                <p>
                  Select your preferred programming language to get started.
                </p>
                {/* Technology selection will go here */}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right panel - Code preview */}
        <div className="w-1/2 overflow-y-auto">
          <Card className="h-full">
            <CardContent className="space-y-6 py-4">
              <div className="space-y-2">
                <h3 className="text-sm font-medium">Task</h3>
                <Code
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
                  language="typescript"
                  value={`// Trigger your task
await task.run({
  name: 'World'
});`}
                  showLineNumbers
                />
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </BasicLayout>
  );
}
