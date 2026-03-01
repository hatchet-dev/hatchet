# frozen_string_literal: true

require 'hatchet-sdk'
require_relative 'mock_scraper'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

SCRAPE_WF = HATCHET.workflow(name: 'ScrapeUrl')
PROCESS_WF = HATCHET.workflow(name: 'ProcessContent')

# > Step 01 Define Scrape Task
SCRAPE_WF.task(:scrape_url, execution_timeout: '2m', retries: 2) do |input, _ctx|
  mock_scrape(input['url'])
end
# !!

# > Step 02 Process Content
PROCESS_WF.task(:process_content) do |input, _ctx|
  mock_extract(input['content'])
end
# !!

# > Step 03 Cron Workflow
CRON_WF = HATCHET.workflow(name: 'WebScrapeWorkflow', on_crons: ['0 */6 * * *'])

CRON_WF.task(:scheduled_scrape) do |_input, _ctx|
  urls = %w[
    https://example.com/pricing
    https://example.com/blog
    https://example.com/docs
  ]

  results = urls.map do |url|
    scraped = SCRAPE_WF.run('url' => url)
    processed = PROCESS_WF.run('url' => url, 'content' => scraped['content'])
    { 'url' => url }.merge(processed)
  end
  { 'refreshed' => results.size, 'results' => results }
end
# !!

def main
  # > Step 04 Run Worker
  worker = HATCHET.worker('web-scraping-worker', slots: 5, workflows: [SCRAPE_WF, PROCESS_WF, CRON_WF])
  worker.start
  # !!
end

main if __FILE__ == $PROGRAM_NAME
