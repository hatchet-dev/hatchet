import { Callout, Tabs } from "nextra/components";
import { snippets } from "@/lib/generated/snippets";
import { Snippet } from "@/components/code";
import PackageManagerInstall from "@/components/PackageManagerInstall";
import UniversalTabs from "@/components/UniversalTabs";

/** Nested tabs: Provider → Language. Wire into scrape_url() / scrapeUrl(). */
export function ScraperIntegrationTabs() {
  return (
    <Tabs
      items={["Firecrawl", "Browserbase", "Playwright", "OpenAI Web Search"]}
    >
      <Tabs.Tab title="Firecrawl">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]}>
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ python: "firecrawl-py" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.python.guides.integrations.scraper_firecrawl
                  .firecrawl_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall
              packages={{ typescript: "@mendable/firecrawl-js" }}
            />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.typescript.guides.integrations.scraper_firecrawl
                  .firecrawl_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">
              Firecrawl Go: use <code>net/http</code> against the{" "}
              <a
                href="https://docs.firecrawl.dev/api-reference"
                target="_blank"
                rel="noopener"
              >
                Firecrawl REST API
              </a>
              .
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">
              Firecrawl Ruby: <code>bundle add firecrawl</code> — see{" "}
              <a
                href="https://docs.firecrawl.dev"
                target="_blank"
                rel="noopener"
              >
                Firecrawl docs
              </a>
              .
            </Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="Browserbase">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]}>
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall
              packages={{ python: "browserbase playwright" }}
            />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.python.guides.integrations.scraper_browserbase
                  .browserbase_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall
              packages={{ typescript: "@browserbasehq/sdk playwright" }}
            />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.typescript.guides.integrations.scraper_browserbase
                  .browserbase_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">
              Browserbase Go: connect via <code>chromedp</code> using the
              session CDP URL. See{" "}
              <a
                href="https://docs.browserbase.com"
                target="_blank"
                rel="noopener"
              >
                Browserbase docs
              </a>
              .
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">
              Browserbase Ruby: use Playwright via{" "}
              <code>playwright-ruby-client</code>. See{" "}
              <a
                href="https://docs.browserbase.com"
                target="_blank"
                rel="noopener"
              >
                Browserbase docs
              </a>
              .
            </Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="Playwright">
        <UniversalTabs items={["Python", "TypeScript", "Go", "Ruby"]}>
          <Tabs.Tab title="Python">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ python: "playwright" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.python.guides.integrations.scraper_playwright
                  .playwright_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="TypeScript">
            <p className="mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Install
            </p>
            <PackageManagerInstall packages={{ typescript: "playwright" }} />
            <p className="mt-3 mb-2 text-sm text-neutral-600 dark:text-neutral-400 font-medium">
              Usage
            </p>
            <Snippet
              src={
                snippets.typescript.guides.integrations.scraper_playwright
                  .playwright_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">
              Playwright Go: use{" "}
              <code>go get github.com/playwright-community/playwright-go</code>.
              See{" "}
              <a
                href="https://github.com/playwright-community/playwright-go"
                target="_blank"
                rel="noopener"
              >
                playwright-go
              </a>
              .
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">
              Playwright Ruby: <code>bundle add playwright-ruby-client</code>.
              See{" "}
              <a
                href="https://playwright-ruby-client.vercel.app/"
                target="_blank"
                rel="noopener"
              >
                docs
              </a>
              .
            </Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
      <Tabs.Tab title="OpenAI Web Search">
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
                snippets.python.guides.integrations.scraper_openai
                  .open_ai_web_search_usage
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
                snippets.typescript.guides.integrations.scraper_openai
                  .open_ai_web_search_usage
              }
            />
          </Tabs.Tab>
          <Tabs.Tab title="Go">
            <Callout type="info">
              OpenAI Go: <code>go get github.com/sashabaranov/go-openai</code> —
              use the Responses API with <code>web_search</code> tool.
            </Callout>
          </Tabs.Tab>
          <Tabs.Tab title="Ruby">
            <Callout type="info">
              OpenAI Ruby: <code>bundle add openai</code> — use the Responses
              API with <code>web_search</code> tool.
            </Callout>
          </Tabs.Tab>
        </UniversalTabs>
      </Tabs.Tab>
    </Tabs>
  );
}

export default ScraperIntegrationTabs;
