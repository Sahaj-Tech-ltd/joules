import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

test.describe('Meal logging', () => {
  test('log page loads with heading', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/log');
    await expect(page.locator('h1', { hasText: 'Log a Meal' })).toBeVisible();
  });

  test('log page shows manual food add button', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/log');
    await expect(page.locator('button', { hasText: 'Manual' })).toBeVisible();
  });

  test('log page shows food search input', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/log');
    await expect(page.locator('input[placeholder="Search foods database..."]')).toBeVisible();
  });
});
