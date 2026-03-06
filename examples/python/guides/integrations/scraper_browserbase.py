# Third-party integration example - requires: pip install playwright browserbase
# See: /guides/web-scraping

import os

from browserbase import Browserbase
from playwright.async_api import async_playwright

bb = Browserbase(api_key=os.environ["BROWSERBASE_API_KEY"])


# > Browserbase usage
async def scrape_url(url: str) -> dict:
    session = bb.sessions.create(project_id=os.environ["BROWSERBASE_PROJECT_ID"])
    async with async_playwright() as pw:
        browser = await pw.chromium.connect_over_cdp(session.connect_url)
        page = browser.contexts[0].pages[0]
        await page.goto(url)
        content = await page.content()
        await browser.close()
    return {"url": url, "content": content}
