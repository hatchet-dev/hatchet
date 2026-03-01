import { Callout, Tabs } from "nextra/components";
import { snippets } from "@/lib/generated/snippets";
import { Snippet } from "@/components/code";
import PackageManagerInstall from "@/components/PackageManagerInstall";
import UniversalTabs from "@/components/UniversalTabs";

/** Nested tabs: Provider → Language. Wire into get_llm_service() / LLMService.generate(). */
export function LLMIntegrationTabs() {
  return (
    <Tabs items={["OpenAI", "Anthropic", "Groq", "Vercel AI SDK", "Ollama"]}>
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
              src={snippets.python.guides.integrations.llm_openai.open_ai_usage}
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
                snippets.typescript.guides.integrations.llm_openai.open_ai_usage
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
              src={snippets.go.guides.integrations.llm_openai.open_ai_usage}
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
              src={snippets.ruby.guides.integrations.llm_openai.open_ai_usage}
            />
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="Anthropic">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]} variant="hidden">
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ python: "anthropic" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.python.guides.integrations.llm_anthropic
                  .anthropic_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall
              packages={{ typescript: "@anthropic-ai/sdk" }}
            />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.typescript.guides.integrations.llm_anthropic
                  .anthropic_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">
              Anthropic Go:{" "}
              <code>go get github.com/anthropics/anthropic-sdk-go</code> — wire{" "}
              <code>messages.Create()</code> into your complete function.
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">
              Anthropic Ruby: <code>bundle add anthropic</code> — wire the
              client into your complete function.
            </Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="Groq">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]} variant="hidden">
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ python: "groq" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={snippets.python.guides.integrations.llm_groq.groq_usage}
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ typescript: "groq-sdk" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={snippets.typescript.guides.integrations.llm_groq.groq_usage}
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">
              Groq: use <code>net/http</code> against{" "}
              <code>api.groq.com/openai/v1/chat/completions</code>. See{" "}
              <a
                href="https://console.groq.com/docs"
                target="_blank"
                rel="noopener"
              >
                Groq docs
              </a>
              .
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">
              Groq Ruby: <code>bundle add groq</code> or use HTTP client. See{" "}
              <a
                href="https://console.groq.com/docs"
                target="_blank"
                rel="noopener"
              >
                Groq docs
              </a>
              .
            </Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="Vercel AI SDK">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]} variant="hidden">
          <Tabs.Tab title="Python">
            <Callout type="info">
              Vercel AI SDK is JavaScript/TypeScript only. Use OpenAI,
              Anthropic, or Groq SDK directly.
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall
              packages={{ typescript: "ai @ai-sdk/openai" }}
            />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.typescript.guides.integrations.llm_vercel_ai_sdk
                  .vercel_ai_sdk_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">
              Vercel AI SDK is JavaScript/TypeScript only.
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">
              Vercel AI SDK is JavaScript/TypeScript only.
            </Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="Ollama">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]} variant="hidden">
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ python: "ollama" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={snippets.python.guides.integrations.llm_ollama.ollama_usage}
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <Callout type="info">
              Use <code>fetch</code> to{" "}
              <code>http://localhost:11434/api/chat</code>. See{" "}
              <a
                href="https://github.com/ollama/ollama/blob/main/docs/api.md"
                target="_blank"
                rel="noopener"
              >
                Ollama API
              </a>
              .
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">
              Use <code>net/http</code> to{" "}
              <code>http://localhost:11434/api/chat</code>. See{" "}
              <a
                href="https://github.com/ollama/ollama/blob/main/docs/api.md"
                target="_blank"
                rel="noopener"
              >
                Ollama API
              </a>
              .
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">
              Use <code>Net::HTTP</code> to{" "}
              <code>http://localhost:11434/api/chat</code>. See{" "}
              <a
                href="https://github.com/ollama/ollama/blob/main/docs/api.md"
                target="_blank"
                rel="noopener"
              >
                Ollama API
              </a>
              .
            </Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
    </Tabs>
  );
}

export default LLMIntegrationTabs;
