"""Mock scraper - no external API dependencies."""

from datetime import datetime, timezone


def mock_scrape(url: str) -> dict:
    return {
        "url": url,
        "title": f"Page: {url}",
        "content": f"Mock scraped content from {url}. In production, use Firecrawl, Browserbase, or Playwright here.",
        "scraped_at": datetime.now(timezone.utc).isoformat(),
    }


def mock_extract(content: str) -> dict:
    return {
        "summary": content[:80],
        "word_count": str(len(content.split())),
    }
