import type { Page } from '@playwright/test';

export async function loginAsAdmin(page: Page) {
  await page.goto('/login');
  await page.fill('#email', 'admin@joules.local');
  await page.fill('#password', 'asdfghjk2003@P');
  await page.click('button[type="submit"]');
  await page.waitForURL('**/dashboard', { timeout: 10000 });
}
