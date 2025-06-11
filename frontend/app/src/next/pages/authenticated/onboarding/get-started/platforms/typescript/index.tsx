import { Badge } from '@/components/ui/badge';
import { CodeHighlighter } from '@/components/ui/code-highlighter';
import { OnboardingInterface } from '../_onboarding.interface';

const TypescriptSetup: typeof typescriptOnboarding.setup = ({
  existingProject,
}) => (
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
            code={'mkdir hatchet-tutorial && cd hatchet-tutorial'}
            copy
          />
        </div>
        <div>
          <h3 className="text-xl font-semibold mb-2">Init a new npm project</h3>
          <CodeHighlighter
            language="plaintext"
            className="text-sm"
            wrapLines={false}
            code={'npm init -y'}
            copy
          />
        </div>
      </>
    )}
    <div>
      <h3 className="text-xl font-semibold mb-2">
        Install Hatchet and dev dependencies
      </h3>
      <CodeHighlighter
        language="plaintext"
        className="text-sm"
        wrapLines={false}
        code={
          existingProject
            ? `npm i @hatchet-dev/typescript-sdk
npm i typescript @types/node ts-node dotenv --save-dev`
            : `npm i @hatchet-dev/typescript-sdk
npm i ts-node dotenv --save-dev`
        }
        copy
      />
      <p className="mt-2">
        We also use dotenv to load the environment variables from a .env file.
        This isn't required, and you can use your own method to load environment
        variables.
      </p>
    </div>
    {!existingProject && (
      <div>
        <h3 className="text-xl font-semibold mb-2">
          Setup your TypeScript configuration
        </h3>
        <p className="mb-2">
          Copy the following code into a new file called{' '}
          <Badge variant="secondary">tsconfig.json</Badge> in your project root
          directory.
        </p>
        <CodeHighlighter
          language="json"
          className="text-sm"
          wrapLines={false}
          code={`{
    "include": ["**/*.ts"],
    "exclude": ["./dist"],
    "compilerOptions": {
      "target": "es2016",
      "module": "commonjs",
      "baseUrl": ".",
      "allowJs": true,
      "declaration": true,
      "outDir": "./dist",
      "esModuleInterop": true,
      "forceConsistentCasingInFileNames": true,
      "strict": true,
      "skipLibCheck": true
    }
  }`}
          copy
        />
      </div>
    )}
    <div>
      <h3 className="text-xl font-semibold mb-2">Define your first workflow</h3>
      <p className="mb-2">
        Copy the following code into a new file called{' '}
        <Badge variant="secondary">first-workflow.ts</Badge> in your project
        root directory.
      </p>
      <CodeHighlighter
        language="typescript"
        code={`import Hatchet, { Workflow } from "@hatchet-dev/typescript-sdk";
import dotenv from "dotenv";

dotenv.config();

const hatchet = Hatchet.init();

const workflow: Workflow = {
  id: "first-workflow",
  description: "This is my first workflow",
  on: {
    event: "tutorial:create",
  },
  steps: [
    {
      name: "first-step",
      run: async (ctx) => {
        console.log(
          "Congratulations! You've successfully triggered your first workflow run! 🎉",
        );

        return {
          result: "success!",
        };
      },
    },
  ],
};

async function main() {
  const worker = await hatchet.worker("tutorial-worker");
  await worker.registerWorkflow(workflow);
  worker.start();
}

main();`}
        copy
      />
    </div>
    <div>
      <h3 className="text-xl font-semibold mb-2">
        Add a script to start your worker
      </h3>
      <p className="mb-2">
        Add the following worker script to the scripts section of your{' '}
        <Badge variant="secondary">package.json</Badge> file.
      </p>
      <CodeHighlighter
        language="json"
        className="text-sm"
        wrapLines={false}
        copyCode={`"worker": "ts-node first-workflow.ts"`}
        code={`{
    // ... the rest of your package.json
    "scripts": {
      // ... other scripts
      "worker": "ts-node first-workflow.ts"
    }
  }`}
        copy
      />
      <p className="mt-4">
        Your project is now ready to rock! Continue to the next step to generate
        your Hatchet auth token and then start the worker.
      </p>
    </div>
  </div>
);

const TypescriptWorker: typeof typescriptOnboarding.worker = () => (
  <div>
    <p className="mb-2">
      Your TypeScript application is now set up. To start your worker, run the
      following command in your terminal:
    </p>
    <CodeHighlighter
      language="plaintext"
      className="text-sm"
      wrapLines={false}
      code={'npm run worker'}
      copy
    />
  </div>
);

export const typescriptOnboarding: OnboardingInterface = {
  setup: TypescriptSetup,
  worker: TypescriptWorker,
};
