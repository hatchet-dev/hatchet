package main

import (
	"log"
	"regexp"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

type ScrapeInput struct {
	URL string `json:"url"`
}

type ProcessInput struct {
	URL     string `json:"url"`
	Content string `json:"content"`
}

const scrapeRateLimitKey = "scrape-rate-limit"

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Define Scrape Task
	scrapeTask := client.NewStandaloneTask("scrape-url", func(ctx hatchet.Context, input ScrapeInput) (map[string]interface{}, error) {
		result := MockScrape(input.URL)
		return map[string]interface{}{
			"url": result.URL, "title": result.Title,
			"content": result.Content, "scraped_at": result.ScrapedAt,
		}, nil
	}, hatchet.WithRetries(2))
	// !!

	// > Step 02 Process Content
	linkRe := regexp.MustCompile(`https?://[^\s<>"']+`)
	processTask := client.NewStandaloneTask("process-content", func(ctx hatchet.Context, input ProcessInput) (map[string]interface{}, error) {
		links := linkRe.FindAllString(input.Content, -1)
		summary := input.Content
		if len(summary) > 200 {
			summary = summary[:200]
		}
		wordCount := len(strings.Fields(input.Content))
		return map[string]interface{}{
			"summary": strings.TrimSpace(summary), "word_count": wordCount, "links": links,
		}, nil
	})
	// !!

	// > Step 03 Cron Workflow
	cronWf := client.NewWorkflow("WebScrapeWorkflow", hatchet.WithWorkflowCron("0 */6 * * *"))

	cronWf.NewTask("scheduled-scrape", func(ctx hatchet.Context, input map[string]interface{}) (map[string]interface{}, error) {
		urls := []string{
			"https://example.com/pricing",
			"https://example.com/blog",
			"https://example.com/docs",
		}

		results := []map[string]string{}
		for _, url := range urls {
			scrapedResult, err := scrapeTask.Run(ctx, ScrapeInput{URL: url})
			if err != nil {
				return nil, err
			}
			var scraped map[string]interface{}
			if err := scrapedResult.Into(&scraped); err != nil {
				return nil, err
			}
			processedResult, err := processTask.Run(ctx, ProcessInput{URL: url, Content: scraped["content"].(string)})
			if err != nil {
				return nil, err
			}
			var processed map[string]string
			if err := processedResult.Into(&processed); err != nil {
				return nil, err
			}
			results = append(results, processed)
		}
		return map[string]interface{}{"refreshed": len(results), "results": results}, nil
	})
	// !!

	// > Step 04 Rate Limited Scrape
	units := 1
	rateLimitedScrapeTask := client.NewStandaloneTask("rate-limited-scrape", func(ctx hatchet.Context, input ScrapeInput) (map[string]interface{}, error) {
		result := MockScrape(input.URL)
		return map[string]interface{}{
			"url": result.URL, "title": result.Title,
			"content": result.Content, "scraped_at": result.ScrapedAt,
		}, nil
	}, hatchet.WithRetries(2), hatchet.WithRateLimits(&types.RateLimit{
		Key:   scrapeRateLimitKey,
		Units: &units,
	}))
	// !!

	// > Step 05 Run Worker
	err = client.RateLimits().Upsert(features.CreateRatelimitOpts{
		Key:      scrapeRateLimitKey,
		Limit:    10,
		Duration: types.Minute,
	})
	if err != nil {
		log.Fatalf("failed to upsert rate limit: %v", err)
	}

	worker, err := client.NewWorker("web-scraping-worker",
		hatchet.WithWorkflows(scrapeTask, processTask, cronWf, rateLimitedScrapeTask),
		hatchet.WithSlots(5),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
	// !!
}
