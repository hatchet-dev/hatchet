package main

import "time"

type ScrapeResult struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	ScrapedAt string `json:"scraped_at"`
}

func MockScrape(url string) ScrapeResult {
	return ScrapeResult{
		URL:       url,
		Title:     "Page: " + url,
		Content:   "Mock scraped content from " + url + ". In production, use Firecrawl, Browserbase, or Playwright here.",
		ScrapedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func MockExtract(content string) map[string]string {
	summary := content
	if len(summary) > 80 {
		summary = summary[:80]
	}
	words := 0
	for _, c := range content {
		if c == ' ' {
			words++
		}
	}
	return map[string]string{
		"summary":    summary,
		"word_count": string(rune(words + 1)),
	}
}
