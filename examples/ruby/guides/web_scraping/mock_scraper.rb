# frozen_string_literal: true

require 'time'

def mock_scrape(url)
  {
    'url' => url,
    'title' => "Page: #{url}",
    'content' => "Mock scraped content from #{url}. In production, use Firecrawl, Browserbase, or Playwright here.",
    'scraped_at' => Time.now.utc.iso8601
  }
end

def mock_extract(content)
  {
    'summary' => content[0, 80],
    'word_count' => content.split.size.to_s
  }
end
