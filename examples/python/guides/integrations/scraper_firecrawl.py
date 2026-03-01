# Third-party integration example - requires: pip install firecrawl-py
# See: /guides/web-scraping

import os
from firecrawl import FirecrawlApp

firecrawl = FirecrawlApp(api_key=os.environ["FIRECRAWL_API_KEY"])


# > Firecrawl usage
def scrape_url(url: str) -> dict:
    result = firecrawl.scrape_url(url, params={"formats": ["markdown"]})
    return {
        "url": url,
        "content": result["markdown"],
        "metadata": result.get("metadata", {}),
    }
