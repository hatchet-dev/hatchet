# frozen_string_literal: true

require "hatchet-sdk"
require_relative "mock_scraper"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

SCRAPE_WF = HATCHET.workflow(name: "ScrapeUrl")
PROCESS_WF = HATCHET.workflow(name: "ProcessContent")

# > Step 01 Define Scrape Task
SCRAPE_WF.task(:scrape_url, execution_timeout: "2m", retries: 2) do |input, _ctx|
  mock_scrape(input["url"])
end
# !!

# > Step 02 Process Content
PROCESS_WF.task(:process_content) do |input, _ctx|
  content = input["content"]
  links = content.scan(%r{https?://[^\s<>"']+})
  summary = content[0, 200].strip
  word_count = content.split.size
  { "summary" => summary, "word_count" => word_count, "links" => links }
end
# !!

# > Step 03 Cron Workflow
CRON_WF = HATCHET.workflow(name: "WebScrapeWorkflow", on_crons: ["0 */6 * * *"])

CRON_WF.task(:scheduled_scrape) do |_input, _ctx|
  urls = %w[
    https://example.com/pricing
    https://example.com/blog
    https://example.com/docs
  ]

  results = urls.map do |url|
    scraped = SCRAPE_WF.run("url" => url)
    processed = PROCESS_WF.run("url" => url, "content" => scraped["content"])
    { "url" => url }.merge(processed)
  end
  { "refreshed" => results.size, "results" => results }
end
# !!

# > Step 04 Rate Limited Scrape
SCRAPE_RATE_LIMIT_KEY = "scrape-rate-limit"

RATE_LIMITED_WF = HATCHET.workflow(name: "RateLimitedScrape")

RATE_LIMITED_WF.task(
  :rate_limited_scrape,
  execution_timeout: "2m",
  retries: 2,
  rate_limits: [Hatchet::RateLimit.new(static_key: SCRAPE_RATE_LIMIT_KEY, units: 1)],
) do |input, _ctx|
  mock_scrape(input["url"])
end
# !!

def main
  # > Step 05 Run Worker
  HATCHET.rate_limits.put(SCRAPE_RATE_LIMIT_KEY, 10, :minute)

  worker = HATCHET.worker("web-scraping-worker",
                          slots: 5,
                          workflows: [SCRAPE_WF, PROCESS_WF, CRON_WF, RATE_LIMITED_WF],)
  worker.start
  # !!
end

main if __FILE__ == $PROGRAM_NAME
