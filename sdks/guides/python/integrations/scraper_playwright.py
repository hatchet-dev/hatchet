# Third-party integration example - requires: pip install playwright && playwright install
# See: /guides/web-scraping

from playwright.async_api import async_playwright


# > Playwright usage
async def scrape_url(url: str) -> dict:
    async with async_playwright() as pw:
        browser = await pw.chromium.launch(headless=True)
        page = await browser.new_page()
        await page.goto(url)
        content = await page.content()
        await browser.close()
    return {"url": url, "content": content}
# !!
