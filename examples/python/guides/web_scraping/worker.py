from hatchet_sdk import Context, EmptyModel, Hatchet

try:
    from .mock_scraper import mock_extract, mock_scrape
except ImportError:
    from mock_scraper import mock_extract, mock_scrape

hatchet = Hatchet(debug=True)

scrape_wf = hatchet.workflow(name="ScrapeUrl")
process_wf = hatchet.workflow(name="ProcessContent")


# > Step 01 Define Scrape Task
@scrape_wf.task(execution_timeout="2m", retries=2)
async def scrape_url(input: dict, ctx: Context) -> dict:
    return mock_scrape(input["url"])


# > Step 02 Process Content
@process_wf.task()
async def process_content(input: dict, ctx: Context) -> dict:
    return mock_extract(input["content"])


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
        scraped = await scrape_wf.aio_run(input={"url": url})
        processed = await process_wf.aio_run(input={"url": url, "content": scraped["content"]})
        results.append({"url": url, **processed})
    return {"refreshed": len(results), "results": results}


def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "web-scraping-worker",
        workflows=[scrape_wf, process_wf, cron_wf],
        slots=5,
    )
    worker.start()


if __name__ == "__main__":
    main()
