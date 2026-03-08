// Third-party integration - requires: pnpm add @browserbasehq/sdk playwright
// See: /guides/web-scraping

import Browserbase from '@browserbasehq/sdk';
import { chromium } from 'playwright';

const bb = new Browserbase({ apiKey: process.env.BROWSERBASE_API_KEY! });

// > Browserbase usage
export async function scrapeUrl(url: string) {
  const session = await bb.sessions.create({
    projectId: process.env.BROWSERBASE_PROJECT_ID!,
  });
  const browser = await chromium.connectOverCDP(session.connectUrl);
  const page = browser.contexts()[0].pages()[0];
  await page.goto(url);
  const content = await page.content();
  await browser.close();
  return { url, content };
}
