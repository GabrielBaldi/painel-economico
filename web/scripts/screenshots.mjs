// Captures the README screenshots from a running server:
//   npm run screenshots                     (uses http://localhost:8103)
//   BASE_URL=http://localhost:5103 npm run screenshots
// Needs Playwright's Chromium (npx playwright install chromium).

import { mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const baseURL = process.env.BASE_URL ?? 'http://localhost:8103';
const outDir = fileURLToPath(new URL('../../docs/', import.meta.url));
const comparison = '/?indicador=selic&comparar=ipca-12m&periodo=10a';

const shots = [
  { file: 'screenshot.png', path: comparison, width: 1280, height: 800, colorScheme: 'light' },
  { file: 'screenshot-dark.png', path: comparison, width: 1280, height: 800, colorScheme: 'dark' },
  {
    file: 'screenshot-mobile.png',
    path: comparison,
    width: 375,
    height: 812,
    colorScheme: 'light',
    isMobile: true,
  },
];

await mkdir(outDir, { recursive: true });
const browser = await chromium.launch();
try {
  for (const shot of shots) {
    const page = await browser.newPage({
      viewport: { width: shot.width, height: shot.height },
      colorScheme: shot.colorScheme,
      isMobile: shot.isMobile ?? false,
    });
    await page.goto(baseURL + shot.path);
    // Wait for every card and both chart lines, with no request in flight.
    await page.locator('path.chart-line-b').waitFor();
    await page.waitForFunction(
      () =>
        !document.querySelector('[aria-busy="true"]') &&
        ![...document.querySelectorAll('.card')].some((c) => c.textContent?.includes('Carregando')),
    );
    await page.evaluate(() => document.fonts.ready);
    if (shot.isMobile) {
      // On a phone the chart is below the cards; frame the chart panel.
      await page.locator('.explorer').evaluate((el) => el.scrollIntoView());
    }
    await page.screenshot({ path: outDir + shot.file });
    console.log(`saved docs/${shot.file}`);
    await page.close();
  }
} finally {
  await browser.close();
}
