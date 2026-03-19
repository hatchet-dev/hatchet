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
        <p className="mt-2 mb-3">
          OpenAI's{" "}
          <a
            href="https://platform.openai.com/docs/guides/text-generation"
            target="_blank"
            rel="noopener noreferrer"
          >
            Chat Completions API
          </a>{" "}
          provides access to GPT models for text generation, function calling,
          and structured outputs. It's the most widely adopted LLM API and
          supports streaming, tool use, and JSON mode.
        </p>
        <UniversalTabs
          items={["Python", "TypeScript", "Go", "Ruby"]}
          variant="hidden"
        >
          <Tabs.Tab title="Python">
            <PackageManagerInstall packages={{ python: "openai" }} />
            <Snippet
              src={snippets.python.guides.integrations.llm_openai.open_ai_usage}
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <PackageManagerInstall packages={{ typescript: "openai" }} />
            <Snippet
              src={
                snippets.typescript.guides.integrations.llm_openai.open_ai_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <PackageManagerInstall
              packages={{ go: "github.com/sashabaranov/go-openai" }}
            />
            <Snippet
              src={snippets.go.guides.integrations.llm_openai.open_ai_usage}
            />
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <PackageManagerInstall packages={{ ruby: "openai" }} />
            <Snippet
              src={snippets.ruby.guides.integrations.llm_openai.open_ai_usage}
            />
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="Anthropic">
        <p className="mt-2 mb-3">
          Anthropic's{" "}
          <a
            href="https://docs.anthropic.com/en/docs/build-with-claude/text-generation"
            target="_blank"
            rel="noopener noreferrer"
          >
            Messages API
          </a>{" "}
          powers the Claude family of models, including{" "}
          <code>claude-sonnet</code> and <code>claude-haiku</code>. Claude
          excels at long-context reasoning, careful instruction following, and
          tool use with extended thinking support.
        </p>
        <UniversalTabs
          items={["Python", "TypeScript", "Go", "Ruby"]}
          variant="hidden"
        >
          <Tabs.Tab title="Python">
            <PackageManagerInstall packages={{ python: "anthropic" }} />
            <Snippet
              src={
                snippets.python.guides.integrations.llm_anthropic
                  .anthropic_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <PackageManagerInstall
              packages={{ typescript: "@anthropic-ai/sdk" }}
            />
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
        <p className="mt-2 mb-3">
          <a
            href="https://console.groq.com/docs/overview"
            target="_blank"
            rel="noopener noreferrer"
          >
            Groq
          </a>{" "}
          provides ultra-fast inference for open-source models like Llama and
          Mixtral using custom LPU hardware. Its OpenAI-compatible API makes it
          a drop-in replacement when you need low latency.
        </p>
        <UniversalTabs
          items={["Python", "TypeScript", "Go", "Ruby"]}
          variant="hidden"
        >
          <Tabs.Tab title="Python">
            <PackageManagerInstall packages={{ python: "groq" }} />
            <Snippet
              src={snippets.python.guides.integrations.llm_groq.groq_usage}
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <PackageManagerInstall packages={{ typescript: "groq-sdk" }} />
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
        <p className="mt-2 mb-3">
          The{" "}
          <a
            href="https://sdk.vercel.ai/docs"
            target="_blank"
            rel="noopener noreferrer"
          >
            Vercel AI SDK
          </a>{" "}
          is a TypeScript toolkit that provides a unified interface across
          providers (OpenAI, Anthropic, Google, and more). It includes helpers
          for streaming, tool calls, and structured object generation via{" "}
          <code>generateText</code> and <code>streamText</code>.
        </p>
        <UniversalTabs
          items={["Python", "TypeScript", "Go", "Ruby"]}
          variant="hidden"
        >
          <Tabs.Tab title="Python">
            <Callout type="info">
              Vercel AI SDK is JavaScript/TypeScript only. Use OpenAI,
              Anthropic, or Groq SDK directly.
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <PackageManagerInstall
              packages={{ typescript: "ai @ai-sdk/openai" }}
            />
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
        <p className="mt-2 mb-3">
          <a
            href="https://ollama.com/"
            target="_blank"
            rel="noopener noreferrer"
          >
            Ollama
          </a>{" "}
          runs open-source models locally — no API key required. It supports
          Llama, Mistral, Gemma, and others through a simple REST API on{" "}
          <code>localhost:11434</code>. Ideal for development, air-gapped
          environments, or when you want full control over your model.
        </p>
        <Callout type="info">
          <strong>Prerequisites</strong> — install Ollama, start the server, and
          pull a model before running the examples below:
          <pre className="mt-2 text-sm">
            {`# Install (macOS / Linux)
curl -fsSL https://ollama.com/install.sh | sh

# Start the server (runs on localhost:11434)
ollama serve

# Pull a model (in a separate terminal)
ollama pull llama3.2`}
          </pre>
        </Callout>
        <UniversalTabs
          items={["Python", "TypeScript", "Go", "Ruby"]}
          variant="hidden"
        >
          <Tabs.Tab title="Python">
            <PackageManagerInstall packages={{ python: "ollama" }} />
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
