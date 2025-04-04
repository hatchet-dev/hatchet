import { Badge } from '@/components/ui/badge';
import { CodeHighlighter } from '@/components/ui/code-highlighter';
import { OnboardingInterface } from '../_onboarding.interface';

const GoSetup: typeof goOnboarding.setup = ({ existingProject }) => (
  <div className="space-y-8">
    {existingProject ? (
      <div>
        <h3 className="text-xl font-semibold mb-2">Navigate to your project</h3>
        <p className="mt-2">
          Open a new terminal and cd into your project directory
        </p>
      </div>
    ) : (
      <>
        <div>
          <h3 className="text-xl font-semibold mb-2">
            Create a new project directory and cd into it
          </h3>
          <CodeHighlighter
            language="plaintext"
            className="text-sm"
            wrapLines={false}
            code={'mkdir hatchet-go-tutorial && cd hatchet-go-tutorial'}
            copy
          />
        </div>
        <div>
          <h3 className="text-xl font-semibold mb-2">
            Initialize a new Go module
          </h3>
          <CodeHighlighter
            language="plaintext"
            className="text-sm"
            wrapLines={false}
            code={'go mod init github.com/<yourusername>/hatchet-go-tutorial'}
            copy
          />
        </div>
      </>
    )}
    <div>
      <h3 className="text-xl font-semibold mb-2">
        Install Hatchet SDK and dependencies
      </h3>
      <CodeHighlighter
        language="plaintext"
        className="text-sm"
        wrapLines={false}
        code={'go get github.com/hatchet-dev/hatchet github.com/joho/godotenv'}
        copy
      />
      <p className="mt-2">
        We also use godotenv to load the environment variables from a .env file.
        This isn't required, and you can use your own method to load environment
        variables.
      </p>
    </div>
    <div>
      <h3 className="text-xl font-semibold mb-2">
        Define your first Go workflow
      </h3>
      <p className="mb-2">
        Copy the following code into a new file called{' '}
        <Badge variant="secondary">first_workflow.go</Badge> in your project
        root directory.
      </p>
      <CodeHighlighter
        language="go"
        code={`package main

import (
  "fmt"
  "log"

  "github.com/hatchet-dev/hatchet/pkg/client"
  "github.com/hatchet-dev/hatchet/pkg/worker"
  "github.com/joho/godotenv"
)

type stepOneOutput struct {
  Results string \`json:"results"\`
}

func main() {
  err := godotenv.Load()
  if err != nil {
    panic(err)
  }

  cleanup, err := run()
  if err != nil {
    panic(err)
  }

  if err := cleanup(); err != nil {
    panic(fmt.Errorf("error cleaning up: %w", err))
  }
}

func run() (func() error, error) {
  c, err := client.New()

  if err != nil {
    return nil, fmt.Errorf("error creating client: %w", err)
  }

  w, err := worker.NewWorker(
    worker.WithClient(
      c,
    ),
  )
  if err != nil {
    return nil, fmt.Errorf("error creating worker: %w", err)
  }

  testSvc := w.NewService("test")

  err = testSvc.On(
    worker.Events("tutorial-worker"),
    &worker.WorkflowJob{
      Name:        "first-workflow",
      Description: "This is my first workflow.",
      Steps: []*worker.WorkflowStep{
        worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
          log.Printf("Congratulations! You've successfully triggered your first workflow run! 🎉")
          return &stepOneOutput{Results: "success!"}, nil
        },
        ).SetName("first-step"),
      },
    },
  )
  if err != nil {
    return nil, fmt.Errorf("error registering workflow: %w", err)
  }

  cleanup, err := w.Start()
  if err != nil {
    panic(err)
  }

  return cleanup, nil
}`}
        copy
      />
    </div>
    <div>
      <p className="mt-4">
        Your project is now ready to rock! Continue to the next step to generate
        your Hatchet auth token and then start the worker.
      </p>
    </div>
  </div>
);

const GoWorker: typeof goOnboarding.worker = () => (
  <div>
    <p className="mb-2">
      Your Go application is now set up. To start your worker, run the following
      command in your terminal:
    </p>
    <CodeHighlighter
      language="plaintext"
      className="text-sm"
      wrapLines={false}
      code={'go run first_workflow.go'}
      copy
    />
  </div>
);

export const goOnboarding: OnboardingInterface = {
  setup: GoSetup,
  worker: GoWorker,
};
