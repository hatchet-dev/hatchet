import re

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.rate_limit import RateLimit, RateLimitDuration
from pydantic import BaseModel

from .mock_scraper import mock_scrape

hatchet = Hatchet(debug=True)


class UrlInput(BaseModel):
    url: str


class ContentInput(BaseModel):
    url: str
    content: str


scrape_wf = hatchet.workflow(name="ScrapeUrl", input_validator=UrlInput)
process_wf = hatchet.workflow(name="ProcessContent", input_validator=ContentInput)


# > Step 01 Define Scrape Task
@scrape_wf.task(execution_timeout="2m", retries=2)
async def scrape_url(input: UrlInput, ctx: Context) -> dict:
    return mock_scrape(input.url)


# > Step 02 Process Content
@process_wf.task()
async def process_content(input: ContentInput, ctx: Context) -> dict:
    content = input.content
    links = re.findall(r"https?://[^\s<>\"']+", content)
    summary = content[:200].strip()
    word_count = len(content.split())
    return {"summary": summary, "word_count": word_count, "links": links}


# > Step 03 Cron Workflow
cron_wf = hatchet.workflow(name="WebScrapeWorkflow", on_crons=["0 */6 * * *"])


@cron_wf.task()
async def scheduled_scrape(input: EmptyModel, ctx: Context) -> dict:
    urls = [
        "https://example.com/pricing",
        "https://example.com/blog",
        "https://example.com/docs",
    ]

    results = []
    for url in urls:
        scraped = await scrape_wf.aio_run(input=UrlInput(url=url))
        processed = await process_wf.aio_run(
            input=ContentInput(url=url, content=scraped["content"])
        )
        results.append({"url": url, **processed})
    return {"refreshed": len(results), "results": results}


# > Step 04 Rate Limited Scrape
SCRAPE_RATE_LIMIT_KEY = "scrape-rate-limit"

rate_limited_wf = hatchet.workflow(name="RateLimitedScrape", input_validator=UrlInput)


@rate_limited_wf.task(
    execution_timeout="2m",
    retries=2,
    rate_limits=[RateLimit(static_key=SCRAPE_RATE_LIMIT_KEY, units=1)],
)
async def rate_limited_scrape(input: UrlInput, ctx: Context) -> dict:
    return mock_scrape(input.url)


def main() -> None:
    # > Step 05 Run Worker
    hatchet.rate_limits.put(SCRAPE_RATE_LIMIT_KEY, 10, RateLimitDuration.MINUTE)

    worker = hatchet.worker(
        "web-scraping-worker",
        workflows=[scrape_wf, process_wf, cron_wf, rate_limited_wf],
        slots=5,
    )
    worker.start()


if __name__ == "__main__":
    main()
