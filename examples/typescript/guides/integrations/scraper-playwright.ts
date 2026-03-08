// Third-party integration - requires: pnpm add playwright
// See: /guides/web-scraping

import { chromium } from 'playwright';

// > Playwright usage
export async function scrapeUrl(url: string) {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto(url);
  const content = await page.content();
  await browser.close();
  return { url, content };
}
