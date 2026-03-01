import { Callout, Tabs } from "nextra/components";
import { snippets } from "@/lib/generated/snippets";
import { Snippet } from "@/components/code";
import PackageManagerInstall from "@/components/PackageManagerInstall";
import UniversalTabs from "@/components/UniversalTabs";

/** Nested tabs: Provider → Language. Wire into get_embedding_service().embed(). */
export function EmbeddingIntegrationTabs() {
  return (
    <Tabs items={["OpenAI", "Cohere"]}>
      <Tabs.Tab title="OpenAI">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]}>
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ python: "openai" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.python.guides.integrations.embedding_openai
                  .open_ai_embedding_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ typescript: "openai" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.typescript.guides.integrations.embedding_openai
                  .open_ai_embedding_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall
              packages={{ go: "github.com/sashabaranov/go-openai" }}
            />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.go.guides.integrations.embedding_openai
                  .open_ai_embedding_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ ruby: "openai" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.ruby.guides.integrations.embedding_openai
                  .open_ai_embedding_usage
              }
            />
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="Cohere">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]} variant="hidden">
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ python: "cohere" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.python.guides.integrations.embedding_cohere
                  .cohere_embedding_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ typescript: "cohere-ai" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.typescript.guides.integrations.embedding_cohere
                  .cohere_embedding_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">
              Cohere Go: <code>go get github.com/cohere-ai/cohere-go</code> —
              use <code>Client.Embed()</code>.
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">
              Cohere Ruby: <code>bundle add cohere-ruby</code>.
            </Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
    </Tabs>
  );
}

export default EmbeddingIntegrationTabs;
