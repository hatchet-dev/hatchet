import { Badge } from '@/components/ui/badge';
import { CodeHighlighter } from '@/components/ui/code-highlighter';
import { OnboardingInterface } from '../_onboarding.interface';

const PythonSetup: typeof pythonOnboarding.setup = ({ existingProject }) => (
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
            code={'mkdir hatchet-python-tutorial && cd hatchet-python-tutorial'}
            copy
          />
        </div>
        <div>
          <h3 className="text-xl font-semibold mb-2">
            Create a virtual environment and activate it
          </h3>
          <CodeHighlighter
            language="plaintext"
            className="text-sm"
            wrapLines={false}
            code={'python3 -m venv venv && source venv/bin/activate'}
            copy
          />
        </div>
      </>
    )}
    <div>
      <h3 className="text-xl font-semibold mb-2">
        Install Hatchet SDK and dotenv
      </h3>
      <CodeHighlighter
        language="plaintext"
        className="text-sm"
        wrapLines={false}
        code={'pip install hatchet-sdk python-dotenv'}
        copy
      />
      <p className="mt-2">
        We also use dotenv to load the environment variables from a .env file.
        This isn't required, and you can use your own method to load environment
        variables.
      </p>
    </div>
    <div>
      <h3 className="text-xl font-semibold mb-2">
        Define your first Python workflow
      </h3>
      <p className="mb-2">
        Copy the following code into a new file called{' '}
        <Badge variant="secondary">first_workflow.py</Badge> in your project
        root directory.
      </p>
      <CodeHighlighter
        language="python"
        code={`from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()

first_workflow = hatchet.workflow(name="first-workflow")

@first_workflow.task()
def first_step(input: EmptyModel, ctx: Context) -> dict[str, str]:
    print("Congratulations! You've successfully triggered your first Python workflow run! 🎉")
    return {"result": "success!"}


def main():
    worker = hatchet.worker("tutorial-worker", workflows=[first_workflow])
    worker.start()


if __name__ == "__main__":
    main()`}
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

const PythonWorker: typeof pythonOnboarding.worker = () => (
  <div>
    <h1 className="text-2xl font-bold mb-2">Start your Python worker</h1>
    <p className="mb-2">
      Your Python worker is now set up. To start your worker, run the following
      command in your terminal:
    </p>
    <CodeHighlighter
      language="plaintext"
      className="text-sm"
      wrapLines={false}
      code={'python first_workflow.py'}
      copy
    />
  </div>
);

export const pythonOnboarding: OnboardingInterface = {
  setup: PythonSetup,
  worker: PythonWorker,
};
